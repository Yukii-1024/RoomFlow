**RoomFlow 酒店房态与预订管理系统 — 后端 API 接口设计文档**  
版本 v1.0 · RESTful API · Go + PostgreSQL  
根据 RoomFlow 的项目定位，后端需要同时服务于三个调用方：  
* 顾客前端：查询酒店、查看房态、创建及管理预订。  
* 酒店管理端：管理酒店、房型、房间、订单和房态。  
* AI Agent：通过与前端相同的后端接口完成查询和操作，不需要单独设计一套业务 API。  
整体采用统一的 RESTful API，前端与 Agent 均通过 HTTP 调用后端，业务规则由后端统一执行。  
**一、接口总体规划**  
统一业务接口架构。Agent 作为独立客户端调用同一套 API。  
建议接口统一使用 /api/v1 前缀，认证使用 JWT，数据交换格式为 JSON。  

| 模块      | 接口数量 | 主要功能           |
| ------- | ---- | -------------- |
| 用户与认证   | 5    | 注册、登录、个人信息     |
| 酒店管理    | 6    | 酒店查询、信息维护      |
| 房型管理    | 5    | 房型、价格、配置       |
| 客房与房态   | 6    | 具体房间及房态维护      |
| 可订库存与报价 | 2    | 日期范围查询、价格计算    |
| 预订订单    | 7    | 创建、查询、取消、入住、退房 |
| 统计分析    | 2    | 管理端概览及入住率      |
| 合计      | 33   | 覆盖核心业务闭环       |

以下按模块列出接口、参数及业务行为。  
**二、接口详细设计**  
**1. 用户与认证模块**  

| 方法    | 路径                 | 描述         | 权限  |
| ----- | ------------------ | ---------- | --- |
| POST  | /auth/register     | 注册顾客账号     | 公开  |
| POST  | /auth/login        | 登录并获取 JWT  | 公开  |
| GET   | /users/me          | 获取当前用户资料   | 已登录 |
| PATCH | /users/me          | 修改姓名、电话等信息 | 已登录 |
| PATCH | /users/me/password | 修改登录密码     | 已登录 |

POST /auth/login  
请求：  
```
{
  "email": "user@example.com",
  "password": "password123"
}
```
响应：  
```
{
  "access_token": "eyJhbGciOi...",
  "token_type": "Bearer",
  "expires_in": 7200,
  "user": {
    "id": 1001,
    "name": "张三",
    "role": "customer"
  }
}
```
用户角色建议定义为 customer（顾客）和 manager（酒店管理者）。酒店管理者的酒店访问权限应通过酒店与用户的关联关系判断。  
**2. 酒店模块**  

| 方法    | 路径                       | 描述              | 权限    |
| ----- | ------------------------ | --------------- | ----- |
| GET   | /hotels                  | 酒店列表，支持城市、名称等筛选 | 公开    |
| GET   | /hotels/{hotelId}        | 酒店详情、设施及介绍      | 公开    |
| POST  | /hotels                  | 创建酒店            | 管理者   |
| PATCH | /hotels/{hotelId}        | 修改酒店名称、地址、介绍等   | 酒店管理者 |
| GET   | /manager/hotels          | 获取当前管理者负责的酒店    | 管理者   |
| PATCH | /hotels/{hotelId}/status | 上架或下架酒店         | 酒店管理者 |

GET /hotels  
查询参数：  

| 参数        | 类型     | 说明         |
| --------- | ------ | ---------- |
| city      | string | 酒店所在城市     |
| keyword   | string | 名称关键词      |
| page      | int    | 页码，默认 1    |
| page_size | int    | 每页数量，默认 20 |

酒店列表只展示已上架的酒店。房间是否可预订，由后续日期库存查询决定。  
**3. 房型管理模块**  
房型是酒店的销售单位，例如标准大床房、双床房、景观房。一个房型可以对应多间实体客房。  

| 方法    | 路径                           | 描述           | 权限    |
| ----- | ---------------------------- | ------------ | ----- |
| GET   | /hotels/{hotelId}/room-types | 查询酒店全部公开房型   | 公开    |
| GET   | /room-types/{typeId}         | 查询房型详情       | 公开    |
| POST  | /hotels/{hotelId}/room-types | 新增房型         | 酒店管理者 |
| PATCH | /room-types/{typeId}         | 修改房型、基础价格及配置 | 酒店管理者 |
| PATCH | /room-types/{typeId}/status  | 启用或停用房型      | 酒店管理者 |

POST /hotels/{hotelId}/room-types  
```
{
  "name": "豪华大床房",
  "description": "配备1.8米大床",
  "base_price": "328.00",
  "max_guests": 2,
  "area": 35,
  "amenities": ["wifi", "air_conditioning"],
  "features": ["near_elevator", "city_view"]
}
```
其中 features 表示客观房间特征，便于顾客或 Agent 根据需求进行筛选和比较。  
**4. 客房与房态管理模块**  
实体客房是具体的房间，例如 801、802、803。  

| 方法     | 路径                      | 描述            | 权限    |
| ------ | ----------------------- | ------------- | ----- |
| GET    | /hotels/{hotelId}/rooms | 查询酒店全部实体客房及状态 | 酒店管理者 |
| GET    | /rooms/{roomId}         | 获取单间客房详情      | 酒店管理者 |
| POST   | /hotels/{hotelId}/rooms | 添加实体客房        | 酒店管理者 |
| PATCH  | /rooms/{roomId}         | 修改房号、所属房型及特征  | 酒店管理者 |
| PATCH  | /rooms/{roomId}/status  | 设置维修、停用等运营状态  | 酒店管理者 |
| DELETE | /rooms/{roomId}         | 删除未关联有效订单的客房  | 酒店管理者 |

PATCH /rooms/{roomId}/status  
```
{
  "status": "maintenance",
  "reason": "空调故障"
}
```
运营状态建议为：  
* active：正常运营  
* maintenance：维修中  
* inactive：停用  
房间的运营状态与预订占用状态应分开处理。 一间房在某天是否被占用，由预订记录和日期范围计算，避免直接修改单个状态字段导致订单与房态不一致。  
**5. 可订库存与报价模块**  
这是整个预订系统的核心模块，也是 Agent 最常调用的接口之一。  

| 方法   | 路径                             | 描述                 | 权限 |
| ---- | ------------------------------ | ------------------ | -- |
| GET  | /hotels/{hotelId}/availability | 查询指定日期范围内可预订房型及数量  | 公开 |
| POST | /quotes                        | 根据入住日期、人数、房型计算订单报价 | 公开 |

GET /hotels/{hotelId}/availability  
请求示例：  
```
GET /api/v1/hotels/1/availability?check_in=2026-12-01&check_out=2026-12-03&guests=2
```
响应示例：  
```
{
  "hotel_id": 1,
  "check_in": "2026-12-01",
  "check_out": "2026-12-03",
  "nights": 2,
  "room_types": [
    {
      "type_id": 10,
      "name": "豪华大床房",
      "available_count": 3,
      "max_guests": 2,
      "base_price": "328.00",
      "features": ["city_view"]
    }
  ]
}
```
库存查询需要排除日期重叠的有效预订，以及不可运营的房间。  
POST /quotes  
```
{
  "hotel_id": 1,
  "room_type_id": 10,
  "check_in": "2026-12-01",
  "check_out": "2026-12-03",
  "guests": 2
}
```
响应：  
```
{
  "currency": "CNY",
  "nights": 2,
  "room_rate_total": "656.00",
  "fees": "0.00",
  "total": "656.00"
}
```
报价由后端计算，订单创建时再次校验价格及库存。  
**6. 预订订单模块**  

| 方法   | 路径                              | 描述          | 权限           |
| ---- | ------------------------------- | ----------- | ------------ |
| POST | /bookings                       | 创建预订        | 顾客           |
| GET  | /bookings                       | 查询当前顾客的预订列表 | 顾客           |
| GET  | /bookings/{bookingId}           | 查询预订详情      | 订单所属顾客或酒店管理者 |
| POST | /bookings/{bookingId}/cancel    | 取消订单        | 顾客或酒店管理者     |
| GET  | /manager/bookings               | 管理端查询酒店订单   | 酒店管理者        |
| POST | /bookings/{bookingId}/check-in  | 办理入住并分配房间   | 酒店管理者        |
| POST | /bookings/{bookingId}/check-out | 办理退房        | 酒店管理者        |

POST /bookings  
```
{
  "hotel_id": 1,
  "room_type_id": 10,
  "check_in": "2026-12-01",
  "check_out": "2026-12-03",
  "guests": 2,
  "contact_name": "张三",
  "contact_phone": "13800138000",
  "special_requests": "希望远离电梯"
}
```
响应：  
```
{
  "id": 5001,
  "booking_no": "RF202612010001",
  "status": "confirmed",
  "total_amount": "656.00",
  "room_id": null
}
```
这里采用按房型预订、入住时分配具体房间的设计。  
订单状态建议如下：  
订单状态流转：已确认 → 已入住 → 已退房；未入住的有效订单可按规则取消。  
创建订单时必须在数据库事务中校验并锁定相应库存，防止并发请求造成超售。  
**7. 统计分析模块**  

| 方法  | 路径                            | 描述         | 权限    |
| --- | ----------------------------- | ---------- | ----- |
| GET | /manager/dashboard            | 获取酒店管理首页概览 | 酒店管理者 |
| GET | /manager/statistics/occupancy | 获取指定时间段入住率 | 酒店管理者 |

GET /manager/dashboard?hotel_id=1  
```
{
  "hotel_id": 1,
  "total_rooms": 50,
  "occupied_rooms": 32,
  "available_rooms": 15,
  "unavailable_rooms": 3,
  "today_check_ins": 8,
  "today_check_outs": 6
}
```
统计口径需要统一，例如已预订但尚未入住的房间是否计入当天占用，应根据查询的具体指标区分。  
**三、统一接口规范**  
**1. 响应格式**  
建议所有接口采用统一的响应结构。  
成功：  
```
{
  "code": "OK",
  "message": "success",
  "data": {}
}
```
失败：  
```
{
  "code": "ROOM_NOT_AVAILABLE",
  "message": "所选日期内房型库存不足",
  "data": null
}
```
建议使用以下 HTTP 状态码：  

| 状态码 | 含义            |
| --- | ------------- |
| 200 | 请求成功          |
| 201 | 创建成功          |
| 400 | 请求参数错误        |
| 401 | 未登录或认证失效      |
| 403 | 无权访问          |
| 404 | 资源不存在         |
| 409 | 业务冲突，例如房间已被预订 |
| 500 | 服务器内部错误       |

**2. 关键业务约束**  
日期重叠判断  
对于两个预订区间，满足以下条件即存在重叠：  
```
existing.check_in < requested.check_out
AND existing.check_out > requested.check_in
```
采用左闭右开区间，因此同一天退房和入住可以共存。  
订单权限  
顾客只能查看和操作自己的订单；酒店管理者只能管理自己负责的酒店及订单。后端不能依赖前端传来的用户身份判断权限。  
并发预订  
创建订单需要事务保护。建议通过 PostgreSQL 的行锁或其他并发控制方案，保证同一房型的库存检查与预订写入具有一致性。  
金额与时间  
金额使用 NUMERIC(12,2) 存储，API 中使用十进制字符串。入住、退房日期使用 YYYY-MM-DD，系统记录时间使用带时区的时间戳。  
**四、Agent 接口设计**  
按照 RoomFlow 的架构，Agent 不需要专属业务接口。  
Agent 的工作是将自然语言请求转换为现有 API 调用，再把结果组织成用户容易理解的形式。  
例如用户说：  
帮我找一下下周五入住、住两晚、两个人住的房间，尽量便宜一些。  
Agent 的调用流程为：  
* 理解日期与入住条件 解析入住日期、退房日期、人数，日期有歧义时要求用户确认。   
* GET /hotels/{hotelId}/availability 获取符合日期和人数条件的可订房型。   
* POST /quotes 获取不同房型的实际总价。   
* 向用户展示客观差异 比较价格、面积、景观、是否靠近电梯等属性，由用户选择。   
* POST /bookings 用户明确确认后，调用创建预订接口。   
这里建议额外实现一层 Agent Tool Adapter，将后端 REST API 封装成工具描述，例如：  
```
{
  "name": "search_available_rooms",
  "description": "查询指定酒店在给定日期范围内的可订房型",
  "parameters": {
    "hotel_id": "integer",
    "check_in": "string",
    "check_out": "string",
    "guests": "integer"
  }
}
```
这层只负责参数校验、接口调用和结果转换，不承担酒店业务规则。  
如果未来要让 Agent 具备修改订单、取消订单等能力，仍然调用原有接口，并在执行前获得用户确认。  
**五、开发优先级**  
33 个接口不需要一次性全部完成。建议按照业务闭环分为三个阶段。  

| 阶段      | 实现范围                         | 目标            |
| ------- | ---------------------------- | ------------- |
| P0 核心闭环 | 认证、酒店查询、房型查询、库存查询、报价、创建及取消订单 | 顾客能够完成完整预订    |
| P1 酒店管理 | 房型 CRUD、客房 CRUD、订单管理、入住退房    | 酒店管理者能够维护房态   |
| P2 扩展功能 | 统计分析、筛选优化、Agent 工具适配         | 完成管理体验与自然语言操作 |

**六、建议的后端目录结构**  
结合 Go 项目，可以按业务模块组织：  
```
backend/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── auth/
│   ├── user/
│   ├── hotel/
│   ├── room/
│   ├── availability/
│   ├── booking/
│   ├── statistics/
│   └── middleware/
├── migrations/
├── api/
│   └── openapi.yaml
├── go.mod
└── go.sum
```
每个业务模块内部再按需要划分 Handler、Service、Repository。  
建议将最终接口契约维护在 api/openapi.yaml 中，使用 OpenAPI 3.1 描述请求参数、响应结构、认证要求和错误码。这样前端、后端和 Agent 都能基于同一份规范进行开发。  
当前版本的设计边界： 采用单酒店或多酒店兼容的数据模型、按房型预订、入住时分配具体房间、无在线支付的简化订单流程。  
RoomFlow 作为软件工程课程项目，这套接口足以支撑顾客预订、酒店房态管理和 Agent 自然语言操作三个主要功能。后续如果增加在线支付、订单改期、动态定价等需求，再扩展相应接口即可。  n`
