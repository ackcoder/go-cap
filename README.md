
## Overview

`go-cap` 是 [@cap.js/server](https://github.com/tiagorangel1/cap) 的 Go 语言实现版

> **Notice**  
> 此包内部依赖 `github.com/redis/go-redis/v9`, 若使用 Redis 需要 `v6.2.0` 以上版本  

> **Reference**  
> [Cap](https://github.com/tiagorangel1/cap) 源仓库  
> [cap_go_server](https://github.com/samwafgo/cap_go_server) 另一个 Go 实现  

## Quick Start

项目内引用  
```sh
go get github.com/ackcoder/go-cap
```

业务逻辑中使用
```go
// 注: 创建实例一般放在 service 或 logic 层中
c := gocap.New()

challenge, err := c.CreateChallenge(context.Background())
fmt.Println("测试", challenge, err) //challenge: 待返回给前端组件的质询数据
```

HTTP 服务示例
```go
package main

import (
	"net/http"
	"fmt"

	gocap "github.com/ackcoder/go-cap"
)

func main() {
	c := gocap.New()

	http.HandleFunc("/challenge", c.HandleCreateChallenge())
	http.HandleFunc("/redeem", c.HandleRedeemChallenge())
	http.HandleFunc("/validate", c.HandleValidateToken())

    fmt.Println("go-cap server start...")
    http.ListenAndServe(":8099", nil)
}
```

## Usage

实例创建 `gocap.New()` 可选配置项入参:

**WithStorage(storage Storage)**  
配置自定义存储, 默认是内存存储(sync.Map实现)  
目前内置了 Redis 存储实现, 只需 Redis **v6.2.0** 以上版本  
也可以自行实现 **Storage** 接口  

**WithChallenge(count, size, difficulty int)**  
配置质询数量(default:50)、大小(default:32)、难度(default:4)  

**WithChallengeExpires(expires int)**  
配置质询过期时间, 默认10分钟  

**WithTokenExpires(expires int)**  
配置验证令牌过期时间, 默认20分钟  

**WithTokenVerifyOnce(isOnce bool)**  
配置验证令牌检查次数, 默认一次性, 比较完即删除  

**WithLimiterParams(rps, burst int)**  
配置限流器参数, 默认10次/秒、最大突发50次  
仅在 Handlexxx 方法调用时生效, 每个 Handlexxx 方法独立计算限流  

## License

参考源项目使用 Appache License 2.0, 请参阅 [LICENSE](./LICENSE) 文件
