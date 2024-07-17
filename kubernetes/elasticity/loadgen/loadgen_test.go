package loadgen_test

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"path"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/montanaflynn/stats"
	"github.com/stretchr/testify/assert"

	//	"k8s.io/apimachinery/pkg/api/errors"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const (
	STATS_WINDOW_SIZE        int64 = 10
	CLUSTER_MONITOR_INTERVAL       = 5 * time.Second
)

var RPS int
var DUR time.Duration
var EXP_DUR time.Duration
var ADDR string

func init() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	flag.IntVar(&RPS, "rps", 1, "Number of requests per second to be supplied by client.")
	flag.StringVar(&ADDR, "addr", "127.0.0.1:8080", "Server address.")
	flag.DurationVar(&DUR, "dur", 1*time.Second, "Request spin duration.")
	flag.DurationVar(&EXP_DUR, "exp_dur", 10*time.Second, "Experiment duration.")
}

type Stats struct {
	mu    sync.Mutex
	start int64
	end   int64
	done  bool
	lats  map[int64][]time.Duration
}

func newStats() *Stats {
	return &Stats{
		lats: make(map[int64][]time.Duration),
	}
}

// Convert from time object to ticks (assuming 1 tick = 1s)
func timeToTicks(t time.Time) int64 {
	return t.UnixMilli() / 1000
}

func (s *Stats) startRecording() {
	s.start = timeToTicks(time.Now())
}

func (s *Stats) stopRecording() {
	s.end = timeToTicks(time.Now())
	s.done = true
}

// Record a request completion and the requests's latency
func (s *Stats) addLatency(completionTime time.Time, lat time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Completion time in ticks (since start)
	cTicks := timeToTicks(completionTime) - s.start
	if _, ok := s.lats[cTicks]; !ok {
		s.lats[cTicks] = []time.Duration{}
	}
	s.lats[cTicks] = append(s.lats[cTicks], lat)
}

func roundToHundredth(f float64) float64 {
	return math.Round(f*100.0) / 100.0
}

func (s *Stats) print(t *testing.T) {
	avgLatPerTick := make([]float64, s.end-s.start+2)
	p50LatPerTick := make([]float64, s.end-s.start+2)
	p90LatPerTick := make([]float64, s.end-s.start+2)
	p99LatPerTick := make([]float64, s.end-s.start+2)
	for i := STATS_WINDOW_SIZE; i <= s.end-s.start+1; i++ {
		flats := []float64{}
		// Iterate through the current window of ticks
		for j := i - STATS_WINDOW_SIZE; j < i; j++ {
			// If there were replies recorded during this tick
			if lat, ok := s.lats[j]; ok {
				// Add replies to flat float slice
				for _, l := range lat {
					flats = append(flats, l.Seconds())
				}
			}
		}
		if len(flats) > 0 {
			var err error
			avgLatPerTick[i], err = stats.Mean(flats)
			avgLatPerTick[i] = roundToHundredth(avgLatPerTick[i])
			assert.Nil(t, err, "Err calc mean: %v", err)
			p50LatPerTick[i], err = stats.Percentile(flats, 50.0)
			p50LatPerTick[i] = roundToHundredth(p50LatPerTick[i])
			assert.Nil(t, err, "Err calc p50: %v", err)
			p90LatPerTick[i], err = stats.Percentile(flats, 90.0)
			p90LatPerTick[i] = roundToHundredth(p90LatPerTick[i])
			assert.Nil(t, err, "Err calc p90: %v", err)
			p99LatPerTick[i], err = stats.Percentile(flats, 99.0)
			p99LatPerTick[i] = roundToHundredth(p99LatPerTick[i])
			assert.Nil(t, err, "Err calc p99: %v", err)
		}
	}
	str := "\n=== Raw latency:\n"
	for i := STATS_WINDOW_SIZE; i <= s.end-s.start+1; i++ {
		if l, ok := s.lats[i]; ok {
			str += fmt.Sprintf("%v: %v\n", i, l)
		}
	}
	log.Printf(str)
	str2 := "\n=== Per-tick latency stats:"
	for i := range avgLatPerTick {
		str2 += fmt.Sprintf("\n\t%d mean:%.2f p50:%.2f p90:%.2f p99:%.2f", i, avgLatPerTick[i], p50LatPerTick[i], p90LatPerTick[i], p99LatPerTick[i])
	}
	log.Printf(str2)
	log.Printf("Latency stats over time: &{ window:%v\n\tavg:%v\n\tp50:%v\n\tp90:%v\n\tp99:%v\n}", STATS_WINDOW_SIZE, avgLatPerTick, p50LatPerTick, p90LatPerTick, p99LatPerTick)
}

func (s *Stats) getDebugCtxStr() string {
	return fmt.Sprintf("[t=%v,svc=%v]", timeToTicks(time.Now())-s.start, "wfe")
}

func doRequest(t *testing.T, wg *sync.WaitGroup, stats *Stats, clntXXX *http.Client, baseurl string, id int64) {
	clnt := &http.Client{
		Timeout:   20 * time.Minute,
		Transport: http.DefaultTransport,
	}
	defer wg.Done()
	log.Printf("Req id:%v", id)
	start := time.Now()
	resp, err := clnt.Get(baseurl + "&id=" + strconv.FormatInt(id, 10))
	lat := time.Since(start)
	if assert.Nil(t, err, "Err get (lat:%v): %v", lat, err) {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		assert.Nil(t, err, "Err read resp body: %v", err)
		log.Printf("Req done id:%v body:%v", id, string(body))
		stats.addLatency(time.Now(), lat)
	}
}

func startRequests(t *testing.T, wg *sync.WaitGroup, stats *Stats, cnt *atomic.Int64, clntXXX *http.Client, baseurl string) {
	msBetweenRequests := 1000 * time.Millisecond / time.Duration(RPS)
	clnt := &http.Client{
		Timeout:   20 * time.Minute,
		Transport: http.DefaultTransport,
	}
	clnt.Transport.(*http.Transport).MaxIdleConnsPerHost = 100000
	clnt.Transport.(*http.Transport).MaxIdleConns = 100000
	// Kick of requests in separate goroutines
	for i := 0; i < RPS; i++ {
		id := cnt.Add(1)
		go doRequest(t, wg, stats, clnt, baseurl, id)
		time.Sleep(msBetweenRequests)
	}
}

func newK8sClnt(t *testing.T) *kubernetes.Clientset {
	log.Printf("Build conf")
	// Read config
	config, err := clientcmd.BuildConfigFromFlags("", path.Join(homedir.HomeDir(), ".kube", "config"))
	assert.Nil(t, err, "Err build config: %v", err)
	log.Printf("Build conf done")
	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	assert.Nil(t, err, "Err make clientset: %v", err)
	return clientset
}

func getDeployment(t *testing.T, k8sclnt *kubernetes.Clientset) *appsv1.Deployment {
	dep, err := k8sclnt.AppsV1().Deployments(apiv1.NamespaceDefault).Get(context.TODO(), "spinhttp", metav1.GetOptions{})
	assert.Nil(t, err, "Err get deployment: %v", err)
	return dep
}

func getPods(t *testing.T, k8sclnt *kubernetes.Clientset) *apiv1.PodList {
	pods, err := k8sclnt.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{})
	assert.Nil(t, err, "Err get pods: %v", err)
	return pods
}

func getHPA(t *testing.T, k8sclnt *kubernetes.Clientset) *autoscalingv2.HorizontalPodAutoscaler {
	hpa, err := k8sclnt.AutoscalingV2().HorizontalPodAutoscalers("default").Get(context.TODO(), "spinhttp-autoscale", metav1.GetOptions{})
	assert.Nil(t, err, "Err get pods: %v", err)
	return hpa
}

func getNReplicas(t *testing.T, hpa *autoscalingv2.HorizontalPodAutoscaler) (int32, int32) {
	return hpa.Status.CurrentReplicas, hpa.Status.DesiredReplicas
}

func getUtil(t *testing.T, hpa *autoscalingv2.HorizontalPodAutoscaler) (int32, int32) {
	curUtil := int32(-1)
	targetUtil := int32(-1)
	for _, m := range hpa.Spec.Metrics {
		if m.Type == autoscalingv2.ResourceMetricSourceType && m.Resource.Name == apiv1.ResourceCPU {
			targetUtil = *m.Resource.Target.AverageUtilization
			break
		}
	}
	for _, m := range hpa.Status.CurrentMetrics {
		if m.Type == autoscalingv2.ResourceMetricSourceType && m.Resource.Name == apiv1.ResourceCPU {
			curUtil = *m.Resource.Current.AverageUtilization
			break
		}
	}
	assert.NotEqual(t, int32(-1), targetUtil, "No CPU utilization target reported")
	return curUtil, targetUtil
}

func (s *Stats) logDeploymentStatus(t *testing.T, k8sclnt *kubernetes.Clientset) {
	hpa := getHPA(t, k8sclnt)
	curNReplicas, desNReplicas := getNReplicas(t, hpa)
	curUtil, targetUtil := getUtil(t, hpa)
	log.Printf("%v AvgUtilAutoscaler currentUtil:%v targetUtil:%v, currentNInstances:%v desiredNInstances:%v\n\tConditions:%v", s.getDebugCtxStr(), curUtil, targetUtil, curNReplicas, desNReplicas, hpa.Status.Conditions)
}

func (s *Stats) monitorK8sClusterStatus(t *testing.T, k8sclnt *kubernetes.Clientset) {
	for !s.done {
		s.logDeploymentStatus(t, k8sclnt)
		time.Sleep(CLUSTER_MONITOR_INTERVAL)
	}
}

func TestLoadgen(t *testing.T) {
	log.Printf("Loadgen exp_dur:%v spin_dur:%v rps:%v addr:%v", EXP_DUR, DUR, RPS, ADDR)
	baseurl := "http://" + ADDR + "/spin?dur=" + DUR.String()
	clnt := &http.Client{
		Timeout:   20 * time.Minute,
		Transport: http.DefaultTransport,
	}
	clnt.Transport.(*http.Transport).MaxIdleConnsPerHost = 100000
	clnt.Transport.(*http.Transport).MaxIdleConns = 100000
	k8sclnt := newK8sClnt(t)
	stats := newStats()
	var wg sync.WaitGroup
	var cnt atomic.Int64
	stats.startRecording()
	go stats.monitorK8sClusterStatus(t, k8sclnt)
	start := time.Now()
	for time.Since(start) < EXP_DUR {
		wg.Add(RPS)
		go startRequests(t, &wg, stats, &cnt, clnt, baseurl)
		time.Sleep(1 * time.Second)
	}
	wg.Wait()
	stats.stopRecording()
	stats.print(t)
}
