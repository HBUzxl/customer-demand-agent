package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cdahttp "customer-demand-agent/internal/channel/http"
	"customer-demand-agent/internal/history"
	"customer-demand-agent/internal/llm"
	"customer-demand-agent/internal/memory/longterm"
	"customer-demand-agent/internal/model"
	"customer-demand-agent/internal/review"
)

// TestConsolidateFullChainHTTP P11 真·端到端（可复现）：mock LLM 网关（HTTP
// 边界）+ 真实 buildTaskRunner 装配 + 真实 httpapi 服务 + 真实 wiki 落盘。
// 链路：POST /api/tasks/consolidate（202）→ Runner 执行（真 exec：读观察注记
// → LLM（mock）→ 解析 → UpsertEntry pending_review）→ GET /api/tasks 轮询到
// done → GET /api/memory 条目 pending_review+固化内容并入 → POST
// /api/review/{type}/{title}/approve → GET 条目 verified。
func TestConsolidateFullChainHTTP(t *testing.T) {
	// 1) mock LLM 网关：返回固化 JSON
	calls := 0
	mockLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"{\"summary\":\"银行高优关注大模型训练数据出境，券商关心投研LLM生成合规\",\"tags\":[\"金融\",\"大模型\",\"数据出境\",\"合规\"],\"content\":\"## 固化要点\\n\\n- 银行：大模型训练数据能否出境是高频关切\\n- 券商：投研报告用 LLM 生成的合规性存疑\"}"}}]}`)
	}))
	defer mockLLM.Close()

	// 2) 真实 wiki（带观察注记条目）
	dir := t.TempDir()
	wikiDir := filepath.Join(dir, "wiki")
	_ = os.MkdirAll(wikiDir, 0o755)
	wiki := longterm.NewWikiStore(wikiDir)
	if err := wiki.Load(); err != nil {
		t.Fatal(err)
	}
	content := "---\ntype: industry\ntitle: 固化E2E\nsummary: 待固化\n---\n\n正文。\n\n### 观察\n\n- [high] 银行问过大模型训练数据出境\n- [medium] 券商关心投研 LLM 生成合规"
	if err := wiki.UpsertEntry(&longterm.Entry{Type: "industry", Title: "固化E2E", Content: content, Summary: "待固化"}); err != nil {
		t.Fatal(err)
	}

	// 3) 真实 model manager（指向 mock 网关）
	registry := model.NewRegistry(5 * time.Second)
	cfgModel := model.ModelConfig{Name: "default", Endpoint: mockLLM.URL, APIKey: "k", Model: "m"}
	_ = registry.Register(cfgModel)
	rc := model.RouterConfig{Default: "default"}
	mgr := model.NewManager(llm.NewClient(), registry, &rc)

	// 4) 真实 history + runner + httpapi
	hist, err := history.Open(filepath.Join(dir, "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer hist.Close()
	runner := buildTaskRunner(wiki, mgr, hist)
	rv := review.New(wiki)
	ts := httptest.NewServer(cdahttp.NewMinimal(runner, wiki, rv, hist).Handler())
	defer ts.Close()

	// 5) POST /api/tasks/consolidate → 202
	res, err := http.Post(ts.URL+"/api/tasks/consolidate", "application/json",
		strings.NewReader(`{"type":"industry","title":"固化E2E"}`))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("consolidate 应 202，got %d", res.StatusCode)
	}
	// 6) 轮询任务 done
	deadline := time.Now().Add(5 * time.Second)
	status := ""
	for time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
		res, err := http.Get(ts.URL + "/api/tasks")
		if err != nil {
			t.Fatal(err)
		}
		var out struct {
			Items []struct {
				Type   string `json:"type"`
				Status string `json:"status"`
				Result string `json:"result"`
			} `json:"items"`
		}
		_ = json.NewDecoder(res.Body).Decode(&out)
		res.Body.Close()
		if len(out.Items) > 0 && out.Items[0].Type == "consolidate" {
			status = out.Items[0].Status
			if status != "running" {
				if status != "done" {
					t.Fatalf("任务应 done，got %s", status)
				}
				break
			}
		}
	}
	if status != "done" {
		t.Fatal("任务未在期限内完成")
	}
	if calls == 0 {
		t.Fatal("应真实调用 LLM（mock 网关）")
	}
	// 7) 条目 pending_review + 固化内容并入
	e, _ := wiki.GetEntry("industry", "固化E2E")
	if e == nil || string(e.Status) != "pending_review" {
		t.Fatalf("固化后应 pending_review: %+v", e)
	}
	if !strings.Contains(e.Content, "固化要点") || !strings.Contains(e.Content, "银行") {
		t.Fatalf("固化产出应并入观察内容: %q", e.Content)
	}
	// 8) HTTP 审批 → verified
	res2, err := http.Post(ts.URL+"/api/review/industry/"+url.PathEscape("固化E2E")+"/approve", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("approve 应 200，got %d", res2.StatusCode)
	}
	e2, _ := wiki.GetEntry("industry", "固化E2E")
	if e2 == nil || string(e2.Status) != "verified" {
		t.Fatalf("审批后应 verified: %+v", e2)
	}
}
