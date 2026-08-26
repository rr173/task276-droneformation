// task276-droneformation 无人机编队避碰意图一致性验证服务
//
// 服务入口：--addr 监听地址，--db SQLite 路径，--smoke-test 自检模式（关闭并重新打开数据库验证持久化）。
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"task276-droneformation/internal/httpapi"
	"task276-droneformation/internal/service"
	"task276-droneformation/internal/smoke"
	"task276-droneformation/internal/store"
)

func main() {
	var (
		addr      = flag.String("addr", ":8080", "HTTP 监听地址")
		dbPath    = flag.String("db", "task276-droneformation.db", "SQLite 数据库路径")
		smokeTest = flag.Bool("smoke-test", false, "运行自检后退出")
	)
	flag.Parse()

	if *smokeTest {
		if err := smoke.Run(*dbPath); err != nil {
			log.Fatalf("smoke test failed: %v", err)
		}
		fmt.Println("smoke test passed")
		return
	}

	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer db.Close()

	app := service.NewApp(db)
	h := httpapi.NewHandler(app)
	log.Printf("task276-droneformation listening on %s (db=%s)", *addr, *dbPath)
	log.Fatal(http.ListenAndServe(*addr, h.Router()))
}
