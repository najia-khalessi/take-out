# take-out项目需求

基于代码库结构分析，这是一个基于 Go语言 构建的外卖配送平台，采用 微服务架构 和 Redis消息队列 实现高并发订单处理+秒杀功能

## 功能需求

用户：注册，登录，下单（选择商家，加入购物车，付款），评论，投诉

商家：注册，增删改商品，接单

骑手：注册，登录，接单（近距离派单功能），派单(多单地址min规划功能)

平台功能：推荐功能，派单，3方沟通页面，

程序后台管理员：先不考虑这个

## 技术栈
后端语言: Go
数据库: MySQL + Redis
消息队列: Redis Pub/Sub
并发处理: Goroutine + Channel
认证: JWT Token
容器化: Docker Compose

## 问题

1. 怎么分辨3方身份？
A：根据登录时的选择不同，登录页面添加一个选项，选择自己的身份再进行登陆

2. 如何在app内部实现长时间无操作持续登录（即长时间登录状态如何保持）？登陆状态设置成多久？
A：Token机制，Redis储存，自动续期
Token令牌（通常是JWT） 和 Session会话 是两种常用的用户认证和状态管理机制,它们用于在用户登录后保持用户的登录状态，并在后续请求中验证用户的身份
用户登录成功后生成Token，Token存入Redis并设置过期时间，返回Token给客户端，客户端保存Token用于后续请求

3. 派单功能中的距离问题，直线距离长度怎么计算实现？
A：geohash

4. 抢单大厅：生产端用户，消费端骑手。设计成什么功能？
A：使用Redis消息队列，

用户下单-生成订单id-订单id进消息队列-骑手消费-完

5. IM即时通讯系统好处
A：通过redis的sub/pub进行发布和订阅消息，本质上还是消息队列
  pub/sub其实就是一个信息传递模式，可以在分布式系统中对消息进行解耦合和异步处理，特点是消息发送方和消息接收方不需要直接交互，而是只需要通过一个中间层来进行信息交流即通过消息代理间接通信，这种解耦可以让系统更灵活，组件可以独立的开发和部署。
  而且这种异步多对多通信可以提高系统的性能，特别是在高吞吐量的场景中。

6. HTTP超文本传输协议
A: 用于客户端和服务器之间的通信，定义了客户端如何像服务器发送请求，服务端如何向客户端发送响应，


7. 连接池目的,连接超时情况
A: 避免频繁建立/关闭连接带来的开销，控制最大并发数，提高系统响应速度节省系统资源
连接获取超时：连接池中所有的连接都被占用，应用程序会等待连接释放，可以设置合理的连接获取时间
连接空闲超时：是指连接在空闲状态下可以保持的最大时间，设置合理的空闲超时时间可以避免连接长时间占用资源
异常处理：自动重连

8. 数据库连接池定期监控：时间间隔为什么使用 Ticker 而不是 Sleep？为什么要监控？
A: 更准确的时间控制，sleep可能会有时间漂移，即时间会受到系统调度影响，误差会随着时间推移而累积，Ticker即使执行时间不固定，也能保持固定间隔，而且Ticker 可以被优雅地停止，避免资源泄露
  及时发现连接池耗尽问题，连接泄漏情况，性能瓶颈，异常连接状态，发现后可以以前扩容合理调整连接池参数，减少故障发生

9. Redis：为什么添加商品到redis，利用redis存储？，为什么用Hash存储？
A: 能减少数据库的访问，避免频繁访问数据库，提高响应速度，因为Redis中读取的速度比mysql中更快，有redis存储热点数据会降低数据库压力
   相比与传统的String方法储存完整的JSON更节省空间，string频繁的内存分配和释放导致内存碎片较多，hash所有字段共享同一个 Hash table畏怯内部使用编码，存储方式为连续内存，减少内存碎片，string更新需要分配整个对象的内存hash只需要修改特定字段支持部分更新，存储时的数据结构更清晰

10. 为什么使用Gin框架？
A: 相比于标准库框架，Gin开发效率高

11. 数据存储流程（在HTTP中）
A: 接受请求参数--参数验证--构建Redis查询
  --缓存命中--返回字节流
  --缓存未命中--构建mysql查询--写入Redis缓存--返回数据
  为什么 MySQL 查询后要写入 Redis？保证后续相同请求可以命中缓存，避免重复查询数据库

12. 数据类型的转换
A: 先从URL获取字符串类型的shopid，然后int化，然后转化成字符串形式，形成键值对在redis中查询缓存，缓存命中就在redis中带着键值对以JSON形式的data字节直接返回给前端，如果没有命中就在mysql中进行查找，缓存命中后由于mysql返回值是结构体切片（根据函数返回值变化），所以进行序列化后再存入redis中，最后以json化形式传前端显示

13. 秒杀功能的实现
A: 1.数据库+Redis双写：数据库持久化+Redis缓存预热：将库存信息提前加载到Redis中，提高秒杀时的读取性能
   2.更新Redis库存：Redis的单线程特性和Lua脚本的原子性，使用Lua脚本确保库存检查和减少操作的原子性。因为在高并发场景下，如果分别执行GET和DECR操作，可能会出现竞态条件
   3.防止超卖（Mysql库存）：验证秒杀活动有效性,再通过乐观锁确保库存不会变成负数

14. Lua脚本逻辑：
获取当前库存数量
检查库存是否存在且大于0
如果满足条件，则减少库存并返回1
否则返回0

## 数据库设计
表的字段设计：
    用户：userid，username, userpassword, phone, Address, UserLatitude,UserLongitude
    商家：shopid, shopname, shoppassword
    骑手：riderid, ridername, riderpassword
    商品：productid, productname, productprice
    订单：orderid, userid, shopid, riderid, starttime, endtime，price
    群聊：groupid, userid, shopid, riderid
    聊天信息：messageid, userid，shopid, riderid, content

数据访问流程：
1. 先查 Redis 缓存
2. 缓存未命中再查 MySQL
3. 将 MySQL 数据更新到 Redis

## 代码设计
1. 技术栈选择：
    后端：GO + Gin框架
    数据库：MySQL + Redis
    消息队列：Redis
    实时通讯：Redis Pub/Sub
2. 连接池：groutine, redispool, mysqlpool
3. http请求：
4. 缓存设计：实现 MySQL + Redis 的双写一致性方案

## 预备流程
1. 用户/骑手/商家注册登录
2. 商家上传菜品 insertproduct

## 核心交互流程
1. 用户获取商家列表
2. 用户获取指定商家商品列表
3. 用户下单购买指定商家指定商品
4. 商家接单确认用户订单，生成聊天群id
5. 商家发布跑腿订单
6. 1. 系统订单到骑手抢单
6. 2. 系统派单给指定骑手
7. 1. 骑手获取订单列表，转7.2
7. 2. 骑手确认订单，并且加入聊天群
8. 骑手 商家 用户三方聊天功能
9. 骑手完成订单，订单结束

## 数据流转实例
1. 用户下单: handlers/user.go → services/order_service.go → database/mysql.go
2. 派单给骑手: services/order_service.go → database/redis.go(消息队列) → handlers/rider.go
3. 三方聊天: handlers/chat.go → websocket/hub.go → 广播给相关用户



！！！
我实现的是一套高并发外卖系统的用户中心模块，核心解决三大问题：
安全认证（双Token机制）、数据一致性（MySQL与Redis双写）、高性能访问（缓存优化）
从密码存储（bcrypt）、传输（HTTPS）到令牌管理（JWT+UUID），覆盖全链路安全，用户查询90%请求命中Redis缓存，DB负载降低70%
用户注册流程：
    安全设计：密码经过bcrypt哈希存储，内存中通过defer主动清零敏感数据
              事务中双重校验：先查用户名冲突，再落库（防并发注册漏洞）
    性能优化：注册后用户数据同步写入Redis（Hash结构），后续查询直接走缓存，降低DB压力
双Token认证体系：Token解析时三重验证：签名校验、过期时间、Redis比对（防重放攻击）
    Access Token：JWT存储用户ID，Redis校验有效性（防篡改+主动失效）
    Refresh Token：无状态UUID，Redis设7天过期，避免长期令牌风险
数据层一致性：
    场景      	技术方案	              容错机制
用户更新资料	开启MySQL事务+Redis     HMSet原子更新	Redis失败仅日志告警（最终一致性）
分页查询用户	LIMIT/OFFSET分页+纯DB查询	避免缓存穿透，专用于管理后台

难点1：缓存与数据库的数据不一致
方案：
采用事务内双写（MySQL先写，Redis后写），MySQL回滚时主动清除Redis脏数据
降级策略：
Redis更新失败时记录日志，后续请求回源DB并补偿缓存

难点2：高并发Token校验性能
优化点：
将KEYS命令改为Hash结构存储Token（HGET token:{userID} → O(1)复杂度）
Redis连接池复用（MaxIdle=20），减少TCP建连开销

难点3：面试官常问的“项目难点”应答建议
“在实现双Token机制时，我们发现Refresh Token的安全存储是关键。
最终方案是：将Token与设备指纹绑定（如IP+UserAgent哈希），并限制单用户最多3个活跃Refresh Token，
既保障用户体验，又有效遏制盗用风险。上线后未出现一例Token劫持事件。”


//用户注册-用户更新
db：(事务+数据库连接池+Redis连接池)
  Redis是内存数据库，数据可能会丢失，Mysql是持久化存储，Redis应该作为Mysql的缓存层
        先插入数据库再插入Redis：如果先插入Redis再插入数据库，如果Redis成功DB失败,则Redis和数据库的数据不一致，如果先插入数据库，只影响缓存的性能，可以通过后续访问DB重建缓存，所以一般是先更新数据库再更新缓存，读操作的话就是先读缓存
  Redis存储：（成功失败都没关系）
      使用Redis的HMSET命令，将用户信息存储为哈希表结构，可以结构化存用户信息便于部分字段更新
      ctx：context的一个实例，用来控制请求的生命周期和超时管理，可以避免客户端无限期地等待服务器响应，可以确保在操作完成后及时释放资源，避免资源泄漏

//HTTP注册：用于客户端（浏览器）与服务器之间的数据传输
1.数据变化：
客户端发送HTTP POST请求到服务器，包含一个JSON结构请求体--服务器接收到之后检查是否为POST创建请求 -- json.Newcode将请求体解析到User结构体中 -- bcrypt对密码进行哈希加密string -- 将数据传给函数进行数据库操作 -- 函数中进行数据库储存+Redis存储 -- 插入成功
返回响应 -- 调用w.WriteHeader(http.StatusCreated)设置状态码为201表示资源创建成功 -- 使用json.NewEncoder(user)将用户信息以JSON格式传输给客户端

HTTP登录：
数据：接受客户端POST请求--解析结构体--验证用户密码是否正确:验证函数将此次输入的密码string自动哈希后string与数据库内的哈希密码string进行比较--正确后--生成Token
流程：接受客户端请求--验证请求结构是否与数据结构相同--检查用户凭证（用户名+密码）--生成token令牌用于保持登录状态，并储存到Redis--最后生成JSON响应给客户端--客户端保存返回的token（通常是Cookie）并且再下次发送HTTP请求时在头部携带token

Token：
一种通用术语，用于表示一种令牌或凭证，用于在客户端和服务器之间进行身份验证和授权
定义JWT载荷部分(claims)--定义JWT的对象(token)，确定其算法H256和载荷--从环境变量中获取密钥--使用密钥对JWT进行签名，生成完整的JWT字符串tokenString

JWT作用：
JWT 通过/自包含令牌+数字签名/解决了分布式系统的认证难题，其核心优势在于无状态、跨域友好和灵活的数据承载能力
是一种具体的 Token 格式，它使用 JSON 格式来编码数据，并通过数字签名确保数据的完整性和真实性。JWT 通常用于身份验证和信息交换
通常在客户端存储，不占用服务器空间，方式Cookie，浏览器(localStorage)

如何验证 JWT 是否有效？
验证签名：确保 JWT 的签名部分是有效的，即 JWT 没有被篡改。
验证载荷中的声明：确保 JWT 的载荷中的声明（如过期时间、签发者等）是有效的。
在 Go 中，可以使用 jwt.ParseWithClaims 方法验证 JWT 的有效性

JWT失效了/过期了怎么办？
1.客户端用刷新令牌（Refresh Token）请求新 JWT。
2.服务端验证刷新令牌有效性，生成新 JWT 返回。
3.若刷新令牌失效，则要求重新登录

JWT令牌验证流程：
1.服务器拦截提取JWT
2.签名验证：用相同密钥和算法重新计算签名，对比是否一致
3.声明校验：检查 exp（是否过期）nbf（是否生效）iss（签发者合法性）
4.授权访问：验证通过后，解析 Payload 中的用户角色，决定资源访问权限。

Token中间件：
函数实现了基于JWT和Redis的双重验证机制：JWT保证令牌完整性，Redis实现状态管理（如令牌吊销）
Authorization 头部的主要作用是告诉服务器请求者的身份信息，以便服务器可以验证请求者是否有权限访问特定的资源。
身份验证信息格式：Authorization: Bearer <token>

刷新Token的核心思路是：
登录时同时生成短期有效的access token和长期有效的refresh token
将refresh token也存储到Redis
当access token过期时，用refresh token换取新的access token

Token中间件流程：
提取Token：从请求头的Authorization字段中提取Bearer Token，并去除前缀"Bearer "。
空值检查：若Token为空，立即返回401 Unauthorized错误。
Redis连接：从连接池获取Redis客户端，并通过defer确保使用后归还连接。
JWT解析与验证：使用jwt.ParseWithClaims解析Token
              通过密钥验证签名有效性
              调用claims.Valid()检查声明时效性（如过期时间）
获取用户ID：从JWT声明中提取user_id字段（需将JSON数字类型转为int）。
Redis校验：构造token:{userID}格式的Redis键
            比对Redis存储的Token与请求携带的是否一致

请求放行：所有验证通过后，调用next.ServeHTTP(w, r)执行后续业务逻辑
整体流程：
 graph TD
    A[用户注册] --> B[用户登录]
    B --> C[凭证验证]
    C --> D[生成JWT+存储Redis]
    D --> E[Token验证中间件]
    E --> F[访问受保护资源]

接口顺序：
1. 用户注册接口：路径：POST /api/register
前端传输：通过 HTTPS 提交用户名、密码
后端处理：使用bcryp将密码哈希，存储用户，响应
2. 用户登录接口：路径：POST /api/login
前端传输：提交用户名、密码
后端验证：凭证验证：查询数据库，用 bcrypt.compare() 比对密码哈希值
         生成令牌：JWT Payload：包含用户 ID、角色、签发时间（iat）、过期时间（exp）。
                    签名算法：推荐 HS256（对称）或 RS256（非对称，更安全）
         Redis 存储：将 JWT 与用户权限绑定存入 Redis，Key 格式如 user_token:{userId}，设置自动过期（与 JWT exp 一致），响应
3. Token 验证中间件：拦截需认证的请求
提取 Token：从请求头 Authorization: Bearer <token> 获取 JWT
验证环节：签名校验：用密钥验证 JWT 完整性（防篡改）
          时效检查：验证 exp（过期时间）和 nbf（生效时间）。
          Redis 校验：查询 Redis 中是否存在该 Token，并获取用户权限（实现主动踢出或黑名单功能）。
结果处理：验证通过：将用户信息注入 req.user，进入业务逻辑。
          验证失败：返回 401 Unauthorized（Token 无效）或 403 Forbidden（权限不足）。



shop_db.go
添加商家--查询商家（分页）--查询附近商家（按距离）

shop.go (接口函数)
商家注册--商家登录（商家维持登录状态）--验证凭证--生成Token（JWT）--验证token中间件
获取商家列表--查询商家商品--查询附近商家
添加商品（更新库存）商家接单--商家派单--查询商品列表

rider_db.go
注册骑手（在用户的基础上）--更新骑手

rider.go
申请骑手身份--生成JWT--验证Token中间件

product_db.go
添加商品

product.go
添加商品--更新库存


！！消息队列
order_db.go 


order.go
                  查询订单状态
                       |
用户下单--订单提交--商家接单（创建聊天群）--商家派单

先随机派单附近骑手--无人接单再骑手选单--骑手接单--完成订单

order.go
技术细节：
    发布订单到Redis频道 order_channel
    使用Redis发布订阅模式异步处理订单（rdb.Publish）
    订单数据以JSON格式存储（json.Marshal）
    缓存键格式：order_status_{orderID}

