#!/usr/bin/env python3
"""
Process Alibaba cluster trace data for memory utilization analysis.

Storage uses a directory of parquet files (one per batch), so appending never
requires reading back existing data. Statistics and plotting are computed in
chunks so the full dataset never needs to be held in memory at once.

Usage:
    # Process CSV files and save to parquet directory:
    ./alibaba-cluster-trace-mem.py --input-dir /path/to/csvs

    # Specify output directory for plots:
    ./alibaba-cluster-trace-mem.py --input-dir /path/to/csvs --plot-dir ./plots

    # Incremental mode - process 10 files at a time, appending to existing data:
    ./alibaba-cluster-trace-mem.py --input-dir /path/to/csvs --incremental --n-files 10
    # Run again to process 10 more files (skips already processed files):
    ./alibaba-cluster-trace-mem.py --input-dir /path/to/csvs --incremental --n-files 10

    # Verbose mode - show which files are skipped/processed:
    ./alibaba-cluster-trace-mem.py --input-dir /path/to/csvs --incremental --verbose

    # Skip plot generation (only show statistics):
    ./alibaba-cluster-trace-mem.py --input-dir /path/to/csvs --no-plots

    # Only update parquet directory, skip all statistics (fastest):
    ./alibaba-cluster-trace-mem.py --input-dir /path/to/csvs --incremental --n-files 10 --no-stats

    # Batch workflow: accumulate data first, analyze later
    ./alibaba-cluster-trace-mem.py --input-dir /path/to/csvs --incremental --n-files 50 --no-stats
    ./alibaba-cluster-trace-mem.py --load-from ./alibaba_trace_data --plot-dir ./plots

    # Reset and start fresh in incremental mode:
    ./alibaba-cluster-trace-mem.py --input-dir /path/to/csvs --incremental --reset
"""

import argparse
import glob
import json
import os
import re
import shutil
import subprocess
import sys
from concurrent.futures import ProcessPoolExecutor, as_completed
from pathlib import Path

import matplotlib
matplotlib.use('Agg')  # Use non-interactive backend
import matplotlib.pyplot as plt
import numpy as np
import pandas as pd
import pyarrow.dataset as ds
from tqdm import tqdm

_C_PROGRAM = Path(__file__).resolve().parent.parent / "compute_ptt_c" / "compute_ptt"


# ---------------------------------------------------------------------------
# Metadata helpers (track which CSV files have already been ingested)
# ---------------------------------------------------------------------------

def get_metadata_path(output_dir):
    return os.path.join(output_dir, "_metadata.json")


def load_metadata(output_dir):
    path = get_metadata_path(output_dir)
    if os.path.exists(path):
        with open(path) as f:
            return json.load(f)
    return {"processed_files": [], "next_batch_idx": 0, "total_rows": 0}


def save_metadata(output_dir, metadata):
    os.makedirs(output_dir, exist_ok=True)
    with open(get_metadata_path(output_dir), "w") as f:
        json.dump(metadata, f, indent=2)


# ---------------------------------------------------------------------------
# CSV reading (parallel)
# ---------------------------------------------------------------------------

def read_csv_file(filepath):
    try:
        return pd.read_csv(filepath)
    except Exception as e:
        # tqdm.write is safe to call from worker processes only in the main
        # process; use print here since this runs in a subprocess.
        print(f"Error reading {filepath}: {e}", file=sys.stderr)
        return None


def load_csv_files_parallel(input_dir, num_workers, n_files=None,
                            processed_files=None, verbose=False):
    """
    Load CSV files from input_dir in parallel, skipping already-processed ones.

    Returns:
        (combined_df, list_of_newly_processed_basenames)
        combined_df is None if there are no new files to process.
    """
    if processed_files is None:
        processed_files = set()

    def _csv_key(path):
        m = re.search(r'(\d+)\.csv$', path)
        return int(m.group(1)) if m else path

    all_csv_files = sorted(glob.glob(os.path.join(input_dir, "*.csv")), key=_csv_key)
    if not all_csv_files:
        print(f"No CSV files found in {input_dir}")
        return None, []

    csv_files = []
    skipped_files = []
    for f in all_csv_files:
        (skipped_files if os.path.basename(f) in processed_files else csv_files).append(f)

    skipped_count = len(skipped_files)

    if not csv_files:
        print(f"All {len(all_csv_files)} CSV files have already been processed")
        return None, []

    print(f"Found {len(all_csv_files)} total CSV files")
    if skipped_count > 0:
        print(f"  → Skipping {skipped_count} already processed files")
        if verbose:
            for fname in skipped_files[:10]:
                print(f"       - {fname}")
            if len(skipped_files) > 10:
                print(f"       ... and {len(skipped_files) - 10} more")

    if n_files is not None and n_files > 0:
        csv_files = csv_files[:n_files]
        print(f"  → Processing {len(csv_files)} new files (--n-files limit)")
    else:
        print(f"  → Processing {len(csv_files)} new files")

    print(f"Using {num_workers} parallel workers...")

    dfs = []
    processed_basenames = []

    with ProcessPoolExecutor(max_workers=num_workers) as executor:
        futures = {executor.submit(read_csv_file, f): f for f in csv_files}
        with tqdm(total=len(csv_files), desc="Reading CSV files", unit="file") as pbar:
            for future in as_completed(futures):
                df = future.result()
                filepath = futures[future]
                if df is not None:
                    dfs.append(df)
                    processed_basenames.append(os.path.basename(filepath))
                pbar.update(1)

    if not dfs:
        print("No new data loaded successfully")
        return None, []

    print(f"Concatenating {len(dfs)} DataFrames...")
    combined_df = pd.concat(dfs, ignore_index=True)
    print(f"Newly loaded rows: {len(combined_df)}")
    return combined_df, processed_basenames


# ---------------------------------------------------------------------------
# Parquet directory helpers
# ---------------------------------------------------------------------------

def save_batch_to_parquet_dir(df, output_dir, batch_idx):
    """Write a single batch parquet file with only the three needed columns,
    sorted by msinstanceid so row groups are contiguous per instance."""
    os.makedirs(output_dir, exist_ok=True)
    df = (
        df[["timestamp", "msinstanceid", "memory_utilization"]]
        .dropna(subset=["timestamp", "memory_utilization"])
        .sort_values(["msinstanceid", "timestamp"])
    )
    path = os.path.join(output_dir, f"part-{batch_idx:06d}.parquet")
    df.to_parquet(path, compression="snappy", index=False)
    size_mb = os.path.getsize(path) / (1024 * 1024)
    print(f"  Wrote {len(df):,} rows → {path} ({size_mb:.1f} MB)")
    return path


def parquet_dir_has_data(output_dir):
    return os.path.isdir(output_dir) and bool(
        glob.glob(os.path.join(output_dir, "part-*.parquet"))
    )


# ---------------------------------------------------------------------------
# Sliding-window P2T via C program
# ---------------------------------------------------------------------------

def compute_instance_ptt(output_dir, num_workers, n_parquet_files=None,
                         skip_parquet_files=None, load_ptt=False):
    """
    Invoke the C PTT program, which reads the parquet files directly via DuckDB,
    computes sliding-window max P2T for every instance, and writes a CSV.
    If load_ptt is True, skip computation and load the cached CSV directly.

    Returns:
        (instance_ptt, n_instances)
        instance_ptt: dict msinstanceid -> (max_ptt_1min, max_ptt_5min)
    """
    csv_path = os.path.join(output_dir, "_ptt_output.csv")

    if load_ptt:
        if not os.path.exists(csv_path):
            print(f"Error: cached PTT file not found at {csv_path}")
            sys.exit(1)
        print(f"Loading cached PTT results from {csv_path}")
    else:
        if not _C_PROGRAM.exists():
            print(f"Error: C program not found at {_C_PROGRAM}")
            print(f"  Please run:  cd {_C_PROGRAM.parent} && make")
            sys.exit(1)

        cmd = [str(_C_PROGRAM), output_dir, csv_path, str(num_workers),
               str(n_parquet_files if n_parquet_files is not None else 0),
               str(skip_parquet_files if skip_parquet_files is not None else 0)]

        print(f"Running C PTT program with {num_workers} thread(s)...")
        subprocess.run(cmd, check=True)

    df = pd.read_csv(csv_path)
    instance_ptt = {
        row["msinstanceid"]: (row["max_ptt_1min"], row["max_ptt_5min"])
        for _, row in df.iterrows()
    }

    if not load_ptt:
        try:
            os.remove(csv_path)
        except FileNotFoundError:
            pass

    return instance_ptt, len(instance_ptt)


# ---------------------------------------------------------------------------
# Plotting (reads only the target instance from the dataset)
# ---------------------------------------------------------------------------

def load_instance_data(output_dir, msinstanceid):
    """Load timestamp and memory_utilization for a single instance without
    reading the entire dataset into memory."""
    dataset = ds.dataset(output_dir, format="parquet")
    table = dataset.to_table(
        columns=["msinstanceid", "timestamp", "memory_utilization"],
        filter=ds.field("msinstanceid") == msinstanceid,
    )
    return table.to_pandas()


def plot_memory_utilization(output_dir, msinstanceid, output_prefix,
                            title_suffix="", plot_dir="."):
    instance_data = load_instance_data(output_dir, msinstanceid)

    if instance_data.empty:
        tqdm.write(f"Warning: No data found for msinstanceid {msinstanceid}")
        return

    instance_data["time_minutes"] = instance_data["timestamp"] / 60000
    instance_data = instance_data.sort_values("time_minutes")

    fig, ax = plt.subplots(figsize=(6.4, 3.2))
    ax.plot(instance_data["time_minutes"], instance_data["memory_utilization"],
            linewidth=0.8, alpha=0.8)
    ax.set_xlabel("Time (minutes)", fontsize=10)
    ax.set_ylabel("Memory Utilization", fontsize=10)
    ax.set_title(f"Memory Utilization - msinstanceid: {msinstanceid}{title_suffix}",
                 fontsize=11)
    ax.grid(True, alpha=0.3, linewidth=0.5)
    ax.set_xlim(left=0)
    ax.set_ylim(0, 1.1)

    plt.tight_layout()
    os.makedirs(plot_dir, exist_ok=True)
    output_file = os.path.join(plot_dir, f"{output_prefix}_msinstance_{msinstanceid}.pdf")
    plt.savefig(output_file, format="pdf", dpi=100, bbox_inches="tight")
    plt.close(fig)


# ---------------------------------------------------------------------------
# Statistics orchestration
# ---------------------------------------------------------------------------

def _report_ptt_window(instance_ptt, window_idx, label, file_tag,
                        output_dir, plot_dir, generate_plots):
    entries = [(msid, ptts[window_idx]) for msid, ptts in instance_ptt.items()]
    if not entries:
        print(f"No valid {label} P2T ratios")
        return

    msids  = np.array([e[0] for e in entries])
    ratios = np.array([e[1] for e in entries])

    pct_map = {
        "Median": np.percentile(ratios, 50),
        "P90":    np.percentile(ratios, 90),
        "P95":    np.percentile(ratios, 95),
        "P99":    np.percentile(ratios, 99),
    }

    print(f"\n--- Peak-to-Trough Ratio ({label} sliding windows) ---")
    print(f"Instances analyzed: {len(entries):,}")
    print("Max-ratio-per-instance statistics:")
    for name, val in pct_map.items():
        print(f"  {name:6s}: {val:.6f}x")

    print("Representative instances:")
    plot_specs = []
    for name, val in pct_map.items():
        idx  = np.argmin(np.abs(ratios - val))
        msid = msids[idx]
        print(f"  {name:6s}: {msid}  (ratio: {ratios[idx]:.6f}x)")
        plot_specs.append((msid, f"ptt_{file_tag}_{name.lower()}",
                           f" ({label} P2T {name})"))

    if generate_plots:
        with tqdm(plot_specs, desc=f"{label} P2T plots", unit="plot",
                  dynamic_ncols=True) as pbar:
            for msid, prefix, suffix in pbar:
                pbar.set_postfix(instance=str(msid))
                plot_memory_utilization(output_dir, msid, prefix, suffix, plot_dir)


def calculate_statistics(output_dir, plot_dir=".", generate_plots=True,
                         chunk_size=500_000, num_workers=4, n_parquet_files=None,
                         skip_parquet_files=None, load_ptt=False):
    print("\n" + "=" * 80)
    print("CALCULATING STATISTICS")
    print("=" * 80)

    instance_ptt, total_rows = compute_instance_ptt(
        output_dir, num_workers, n_parquet_files,
        skip_parquet_files=skip_parquet_files, load_ptt=load_ptt
    )

    if total_rows == 0:
        print("Error: no valid rows found")
        return

    _report_ptt_window(instance_ptt, 0, "1-min", "1min",
                       output_dir, plot_dir, generate_plots)
    _report_ptt_window(instance_ptt, 1, "5-min", "5min",
                       output_dir, plot_dir, generate_plots)

    if not generate_plots:
        print("\nPlot generation skipped (--no-plots enabled)")

    print("\n" + "=" * 80)


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(
        description="Process Alibaba cluster trace memory utilization data"
    )
    parser.add_argument("--input-dir",
                        help="Directory containing CSV files to process")
    parser.add_argument("--load-from",
                        help="Analyse an existing parquet directory (skip CSV ingestion)")
    parser.add_argument("--output-dir", default="alibaba_trace_data",
                        help="Directory to store parquet files (default: alibaba_trace_data/)")
    parser.add_argument("--workers", type=int, default=4,
                        help="Parallel workers for CSV reading (default: 4)")
    parser.add_argument("--n-files", type=int, default=None,
                        help="Limit on number of NEW CSV files to process per run")
    parser.add_argument("--n-parquet-files", type=int, default=None,
                        help="Limit on number of parquet files read when computing statistics")
    parser.add_argument("--skip-parquet-files", type=int, default=None,
                        help="Number of parquet files to skip (0-indexed) before reading")
    parser.add_argument("--chunk-size", type=int, default=100_000_000,
                        help="Rows per chunk when scanning parquet for statistics (default: 100_000_000)")
    parser.add_argument("--incremental", action="store_true",
                        help="Append new data to the existing parquet directory")
    parser.add_argument("--reset", action="store_true",
                        help="Delete existing parquet directory and start fresh")
    parser.add_argument("--plot-dir", default=".",
                        help="Directory to save PDF plots (default: current directory)")
    parser.add_argument("--verbose", action="store_true",
                        help="Show detailed file-level processing information")
    parser.add_argument("--no-plots", action="store_true",
                        help="Skip plot generation")
    parser.add_argument("--no-stats", action="store_true",
                        help="Skip statistics (only ingest CSVs and update parquet directory)")
    parser.add_argument("--load-ptt", action="store_true",
                        help="Load PTT results from cached _ptt_output.csv instead of recomputing")

    args = parser.parse_args()

    if args.load_from and args.no_stats:
        print("Error: --load-from with --no-stats has nothing to do")
        sys.exit(1)

    # Determine the parquet directory to work with
    if args.load_from:
        analysis_dir = args.load_from
        if not parquet_dir_has_data(analysis_dir):
            print(f"Error: {analysis_dir} does not exist or contains no parquet files")
            sys.exit(1)
        print(f"Analysing existing parquet dataset at {analysis_dir}")

    elif args.input_dir:
        if not os.path.isdir(args.input_dir):
            print(f"Error: {args.input_dir} is not a valid directory")
            sys.exit(1)

        analysis_dir = args.output_dir

        if args.incremental:
            print("=== Incremental Mode ===")

            if args.reset:
                print("Resetting: removing existing parquet directory and metadata...")
                if os.path.exists(analysis_dir):
                    shutil.rmtree(analysis_dir)
                    print(f"Removed {analysis_dir}/")
                metadata = {"processed_files": [], "next_batch_idx": 0, "total_rows": 0}
                save_metadata(analysis_dir, metadata)
            else:
                metadata = load_metadata(analysis_dir)
                print(f"Previously processed: {len(metadata['processed_files'])} files "
                      f"({metadata.get('total_rows', 0):,} rows)")

            processed_set = set(metadata["processed_files"])
            new_df, newly_processed = load_csv_files_parallel(
                args.input_dir, args.workers, args.n_files, processed_set, args.verbose
            )

            if new_df is None:
                if not parquet_dir_has_data(analysis_dir):
                    print("No data available for analysis")
                    sys.exit(1)
                print("\nNo new files to process — using existing dataset for analysis")
            else:
                # Duplicate safety check
                existing_set = set(metadata["processed_files"])
                dupes = [f for f in newly_processed if f in existing_set]
                if dupes:
                    print(f"WARNING: {len(dupes)} duplicate filenames detected; skipping them")
                    newly_processed = [f for f in newly_processed if f not in existing_set]

                # Append: write the new batch as its own parquet file (no re-read of existing data)
                batch_idx = metadata.get("next_batch_idx", 0)
                save_batch_to_parquet_dir(new_df, analysis_dir, batch_idx)

                metadata["processed_files"].extend(newly_processed)
                metadata["next_batch_idx"] = batch_idx + 1
                metadata["total_rows"] = metadata.get("total_rows", 0) + len(new_df)
                save_metadata(analysis_dir, metadata)

                print(f"\n✓ Processed {len(newly_processed)} new files")
                print(f"✓ Parquet directory: {len(metadata['processed_files'])} total files ingested "
                      f"({metadata['total_rows']:,} rows)")
                if args.verbose:
                    for fname in newly_processed[:10]:
                        print(f"    - {fname}")
                    if len(newly_processed) > 10:
                        print(f"    ... and {len(newly_processed) - 10} more")

        else:
            # Non-incremental: process files and write a single batch
            print("=== Standard Mode ===")
            if parquet_dir_has_data(analysis_dir) and not args.reset:
                print(f"Note: {analysis_dir}/ already exists.")
                print(f"  Use --incremental to append, or --reset to start fresh.")
                print(f"  Proceeding to overwrite...\n")
                shutil.rmtree(analysis_dir)

            new_df, _ = load_csv_files_parallel(
                args.input_dir, args.workers, args.n_files
            )
            if new_df is None:
                print("Failed to load data")
                sys.exit(1)

            save_batch_to_parquet_dir(new_df, analysis_dir, 0)

    else:
        print("Error: Must provide either --input-dir or --load-from")
        parser.print_help()
        sys.exit(1)

    # Statistics
    if args.no_stats:
        print("\nSkipping statistics (--no-stats). Parquet directory is up to date.")
    else:
        if not args.no_plots and args.plot_dir != ".":
            os.makedirs(args.plot_dir, exist_ok=True)
            print(f"\nPlots will be saved to: {os.path.abspath(args.plot_dir)}")

        calculate_statistics(
            analysis_dir,
            plot_dir=args.plot_dir,
            generate_plots=not args.no_plots,
            chunk_size=args.chunk_size,
            num_workers=args.workers,
            n_parquet_files=args.n_parquet_files,
            skip_parquet_files=args.skip_parquet_files,
            load_ptt=args.load_ptt,
        )


if __name__ == "__main__":
    main()
