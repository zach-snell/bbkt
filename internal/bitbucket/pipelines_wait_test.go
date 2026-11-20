package bitbucket

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func waitArgs() WaitPipelineArgs {
	return WaitPipelineArgs{
		GetPipelineArgs: GetPipelineArgs{Workspace: "ws", RepoSlug: "repo", PipelineUUID: "{p}"},
		Interval:        time.Millisecond,
	}
}

func TestWaitPipeline_ReturnsWhenCompleted(t *testing.T) {
	var polls atomic.Int32
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		if polls.Add(1) < 3 {
			_, _ = w.Write([]byte(`{"build_number":7,"state":{"name":"IN_PROGRESS"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"build_number":7,"state":{"name":"COMPLETED","result":{"name":"FAILED"}}}`))
	})

	seen := 0
	args := waitArgs()
	args.OnPoll = func(*Pipeline) error { seen++; return nil }
	p, err := c.WaitPipeline(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if p.State.Result.Name != "FAILED" || polls.Load() != 3 || seen != 3 {
		t.Fatalf("result=%s polls=%d onPoll=%d", p.State.Result.Name, polls.Load(), seen)
	}
}

func TestWaitPipeline_StopsOnPausedStage(t *testing.T) {
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"state":{"name":"IN_PROGRESS","stage":{"name":"PAUSED"}}}`))
	})
	if _, err := c.WaitPipeline(context.Background(), waitArgs()); err != nil {
		t.Fatal(err)
	}
}

func TestWaitPipeline_TimeoutReturnsLastRun(t *testing.T) {
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"build_number":9,"state":{"name":"IN_PROGRESS"}}`))
	})
	args := waitArgs()
	args.Timeout = 20 * time.Millisecond
	p, err := c.WaitPipeline(context.Background(), args)
	if !errors.Is(err, ErrWaitTimeout) {
		t.Fatalf("err = %v, want ErrWaitTimeout", err)
	}
	if p == nil || p.BuildNumber != 9 {
		t.Fatalf("last run = %+v", p)
	}
}

func TestWaitPipeline_ToleratesTransientErrors(t *testing.T) {
	var polls atomic.Int32
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch polls.Add(1) {
		case 1, 2:
			w.WriteHeader(http.StatusBadGateway)
		default:
			_, _ = w.Write([]byte(`{"state":{"name":"COMPLETED"}}`))
		}
	})
	if _, err := c.WaitPipeline(context.Background(), waitArgs()); err != nil {
		t.Fatalf("two failures in a row should be retried: %v", err)
	}
}

func TestWaitPipeline_ContextInterruptsSlowPoll(t *testing.T) {
	release := make(chan struct{})
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		<-release // the poll hangs until the test ends
		_, _ = w.Write([]byte(`{"state":{"name":"COMPLETED"}}`))
	})
	// Registered after the server so it runs first (cleanups are LIFO):
	// srv.Close waits for the blocked handler.
	t.Cleanup(func() { close(release) })
	args := waitArgs()
	args.Timeout = 20 * time.Millisecond
	start := time.Now()
	_, err := c.WaitPipeline(context.Background(), args)
	if !errors.Is(err, ErrWaitTimeout) {
		t.Fatalf("err = %v, want ErrWaitTimeout", err)
	}
	if d := time.Since(start); d > time.Second {
		t.Fatalf("wait returned after %s; the timeout should cut the slow poll short", d)
	}
}

func TestListAllPipelineSteps_FollowsNext(t *testing.T) {
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			_, _ = w.Write([]byte(`{"values":[{"name":"deploy","state":{"name":"COMPLETED","result":{"name":"FAILED"}}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"values":[{"name":"build"}],"next":"https://api.bitbucket.org/2.0/repositories/ws/repo/pipelines/%7Bp%7D/steps?page=2"}`))
	})
	res, err := c.ListAllPipelineSteps(ListPipelineStepsArgs{Workspace: "ws", RepoSlug: "repo", PipelineUUID: "{p}"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Values) != 2 || res.Values[1].Name != "deploy" || res.Next != "" {
		t.Fatalf("got %+v next=%q", res.Values, res.Next)
	}
}

func TestWaitPipeline_GivesUpAfterThreeErrors(t *testing.T) {
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	if _, err := c.WaitPipeline(context.Background(), waitArgs()); err == nil {
		t.Fatal("expected an error after repeated failures")
	}
}
