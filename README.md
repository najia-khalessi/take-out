# take-out项目需求

我们要做一个外卖系统，首先看向市场其他同类竞争产品，先不考虑抢用户
功能上要向其看齐，再考虑用户体验，再考虑我们的产品特色

## 功能需求

用户：注册，登录，下单（选择商家，加入购物车，付款），评论，投诉

商家：注册，增删改商品，接单

骑手：注册，登录，接单（近距离派单功能），派单(多单地址min规划功能)

平台功能：推荐功能，派单，3方沟通页面，

程序后台管理员：先不考虑这个

## 问题

1.怎么分辨3方身份？
A：根据登录时的选择不同，登录页面添加一个选项，选择自己的身份再进行登陆

2.如何再app内部实现长时间无操作持续登录？登陆状态设置成多久？
A：token，代码细节下面代码设计讲

3.派单功能中的距离问题，直线距离长度怎么计算实现？
A：geohash

4.抢单大厅：生产端用户，消费端骑手。设计成什么功能？
A：使用Redis消息队列。

用户下单-生成订单id-订单id进消息队列-骑手消费-完

5.IM即时通讯系统
A：通过redis的sub/pub进行发布和订阅消息，本质上还是消息队列

## 数据库设计 （为什么会想到要用数据库？）
表的字段设计：
    用户：userid，username, userpassword
    商家：shopid, shopname, shoppassword
    骑手：riderid, ridername, riderpassword
    商品：productid, productname, price
    订单：orderid, userid, shopid, riderid, starttime, endtime，allprice
    群聊：groupid, userid, shopid, riderid
    聊天信息：messageid(虽然还是有点离谱), userid，shopid, riderid, context

表的索引设计：
    


## 代码设计
讲出原因，为什么要用这个设计？
1.缓存策略：Read/Write Through（读穿 / 写穿）策略。
2.连接池：groutine, redispool, mysqlpool
3.http请求：
4.

## 预备流程
1. 用户/骑手/商家注册登录
2. 商家上传菜品

核心交互流程
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