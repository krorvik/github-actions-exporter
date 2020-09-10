/*
Main application package
*/
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli"

	"github-actions-exporter/config"
)

var version = "v1.2"

var (
	runnersGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "github_runner_status",
			Help: "runner status",
		},
		[]string{"repo", "os", "name"},
	)

	jobsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "github_job",
			Help: "job status",
		},
		[]string{"repo", "id", "head_branch", "run_number", "event", "status"},
	)

	workflowLatestStatusGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "github_workflow_latest_status",
			Help: "workflow latest status",
		},
		[]string{"repo", "workflow", "head_branch", "event"},
	)

	workflowRunsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "github_workflow_runs",
			Help: "Workflow runs",
		},
		[]string{"repo", "workflow", "id", "url", "created_at", "updated_at", "head_branch", "event"},
	)

)

type runners struct {
	TotalCount int      `json:"total_count"`
	Runners    []runner `json:"runners"`
}

type runner struct {
	Name   string `json:"name"`
	OS     string `json:"os"`
	Status string `json:"status"`
}

type jobsReturn struct {
	TotalCount   int   `json:"total_count"`
	WorkflowRuns []job `json:"workflow_runs"`
}

type job struct {
  ID         int    `json:"id"`
	HeadBranch string `json:"head_branch"`
  RunNumber  int    `json:"run_number"`
	Event      string `json:"event"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	UpdatedAt  string `json:"updated_at"`
}

type workflows struct {
	TotalCount int        `json:"total_count"`
	Workflows  []workflow `json:"workflows"`
}

type workflow struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

type workflowRuns struct {
	TotalCount   int           `json:"total_count"`
	WorkflowRuns []workflowRun `json:"workflow_runs"`
}

type workflowRun struct {
	ID         int    `json:"id"`
	HeadBranch string `json:"head_branch"`
	Event      string `json:"event"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
  URL        string `json:"html_url"`
}

// main init configuration
func main() {
	app := cli.NewApp()
	app.Name = "github-actions-exporter"
	app.Flags = config.NewContext()
	app.Action = runWeb
	app.Version = version

	app.Run(os.Args)
}

// runWeb start http server
func runWeb(ctx *cli.Context) {
	go getRunnersFromGithub()
	go getJobsFromGithub()
	go getWorkflowLatestStatus()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "/metrics")
	})
	http.Handle("/metrics", promhttp.Handler())
	log.Printf("starting exporter with port %v", config.Port)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(config.Port), nil))
}

// init prometheus metrics
func init() {
	prometheus.MustRegister(runnersGauge)
	prometheus.MustRegister(jobsGauge)
	prometheus.MustRegister(workflowLatestStatusGauge)
	prometheus.MustRegister(workflowRunsGauge)
}

func getRunnersFromGithub() {
	client := &http.Client{}

	for {
		for _, repo := range config.Github.Repositories {
			var p runners
			req, _ := http.NewRequest("GET", "https://api.github.com/repos/"+repo+"/actions/runners", nil)
			req.Header.Set("Authorization", "token "+config.Github.Token)
			resp, err := client.Do(req)
			if err != nil {
				log.Fatal(err)
			}
			err = json.NewDecoder(resp.Body).Decode(&p)
			if err != nil {
				log.Fatal(err)
			}
			for _, r := range p.Runners {
				if r.Status == "online" {
					runnersGauge.WithLabelValues(repo, r.OS, r.Name).Set(1)
				} else {
					runnersGauge.WithLabelValues(repo, r.OS, r.Name).Set(0)
				}

			}
		}

		time.Sleep(time.Duration(config.Github.Refresh) * time.Second)
	}
}

func getJobsFromGithub() {
	client := &http.Client{}

	for {
		for _, repo := range config.Github.Repositories {
			var p jobsReturn
			req, _ := http.NewRequest("GET", "https://api.github.com/repos/"+repo+"/actions/runs", nil)
			req.Header.Set("Authorization", "token "+config.Github.Token)
			resp, err := client.Do(req)
			if err != nil {
				log.Fatal(err)
			}
			err = json.NewDecoder(resp.Body).Decode(&p)
			if err != nil {
				log.Fatal(err)
			}
			for _, r := range p.WorkflowRuns {
				var s float64 = 0
				if r.Conclusion == "success" {
					s = 1
				} else if r.Conclusion == "skipped" {
					s = 2
				} else if r.Status == "in_progress" {
					s = 3
				} else if r.Status == "queued" {
					s = 4
				}
				jobsGauge.WithLabelValues(repo, strconv.Itoa(r.ID), r.HeadBranch, strconv.Itoa(r.RunNumber), r.Event, r.Status).Set(s)
			}
		}

		time.Sleep(time.Duration(config.Github.Refresh) * time.Second)
	}
}

func getWorkflowLatestStatus() {
	client := &http.Client{}

	for {
		for _, repo := range config.Github.Repositories {
			var ws workflows
			reqWs, _ := http.NewRequest("GET", "https://api.github.com/repos/"+repo+"/actions/workflows", nil)
			reqWs.Header.Set("Authorization", "token "+config.Github.Token)
			respWs, err := client.Do(reqWs)
			if err != nil {
				log.Fatal(err)
			}
			err = json.NewDecoder(respWs.Body).Decode(&ws)
			if err != nil {
				log.Fatal(err)
			}

			for _, w := range ws.Workflows {
				var wrs workflowRuns
				req, _ := http.NewRequest("GET", "https://api.github.com/repos/"+repo+"/actions/workflows/"+strconv.Itoa(w.ID)+"/runs", nil)
				req.Header.Set("Authorization", "token "+config.Github.Token)
				resp, err := client.Do(req)
				if err != nil {
					log.Fatal(err)
				}
				err = json.NewDecoder(resp.Body).Decode(&wrs)
				if err != nil {
					log.Fatal(err)
				}

				log.Printf("wrs.TotalCount: %d, w.Name: %s", wrs.TotalCount, w.Name)

				if wrs.TotalCount > 0 {
					r := wrs.WorkflowRuns[0]
					var s float64 = 0
					if r.Conclusion == "success" {
						s = 1
					} else if r.Conclusion == "skipped" {
						s = 2
					} else if r.Status == "in_progress" {
						s = 3
					} else if r.Status == "queued" {
						s = 4
					}
					workflowLatestStatusGauge.WithLabelValues(repo, w.Name, r.HeadBranch, r.Event).Set(s)
          for _,  run := range wrs.WorkflowRuns {
            var s float64 = 0
					  if run.Conclusion == "success" {
						  s = 1
					  } else if run.Conclusion == "skipped" {
						  s = 2
					  } else if run.Status == "in_progress" {
						  s = 3
					  } else if run.Status == "queued" {
						  s = 4
					  }
            workflowRunsGauge.WithLabelValues(repo, w.Name, strconv.Itoa(run.ID), run.URL, run.CreatedAt, run.UpdatedAt, run.HeadBranch, run.Event).Set(s)

          }
				}
			}
		}

		time.Sleep(time.Duration(config.Github.Refresh) * time.Second)
	}
}
