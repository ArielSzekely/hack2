
/// This is a simple job to test job start latency.

#include <ray/api.h>
#include <chrono>
#include <unistd.h>

/// common function
int64_t SimpleTask(std::chrono::time_point<std::chrono::high_resolution_clock> spawn_time) {
  auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(std::chrono::high_resolution_clock::now() - spawn_time).count();
  return ms;
}

/// Declare remote function
RAY_REMOTE(SimpleTask);

int main(int argc, char **argv) {
  int N = 5;
  /// initialization
  ray::Init();

  /// common task
  int64_t total_dur_ms = 0;
  std::vector<ray::ObjectRef<int64_t>> task_objs;
  for (int i = 0; i < N; ++i) {
    task_objs.push_back(ray::Task(SimpleTask).Remote(std::chrono::high_resolution_clock::now()));
    auto dur_ms = *(ray::Get(task_objs[i]));
    total_dur_ms += dur_ms;
    std::cout << "start latency = " << dur_ms <<  "ms" << std::endl;
  }
  std::cout << "avg start latency = " << (double) total_dur_ms / (double) N <<  "ms" << std::endl;
  std::cout << "total dur = " << total_dur_ms <<  "ms" << std::endl;

  /// shutdown
  ray::Shutdown();
  return 0;
}
