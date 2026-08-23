// namemerge 植物学名异名归并证据服务入口。
//
// 用法：
//
//	namemerge --addr :8080 --db ./namemerge.db     # 启动 HTTP 服务
//	namemerge --smoke-test                          # 冒烟自检（Docker CMD 默认）
//	namemerge --smoke-test --db ./tmp.db            # 自检并指定数据库路径
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"task186-namemerge/internal/httpapi"
	"task186-namemerge/internal/service"
	"task186-namemerge/internal/smoke"
	"task186-namemerge/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "namemerge.db", "SQLite database path")
	smokeTest := flag.Bool("smoke-test", false, "run smoke test and exit")
	flag.Parse()

	if *smokeTest {
		if err := smoke.Run(*dbPath); err != nil {
			log.Printf("smoke test FAILED: %v", err)
			os.Exit(1)
		}
		return
	}

	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	app := service.NewApp(db)
	srv := &http.Server{
		Addr:              *addr,
		Handler:           httpapi.NewServer(app).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("namemerge listening on %s (db=%s)", *addr, *dbPath)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
	fmt.Println("server stopped")
}
