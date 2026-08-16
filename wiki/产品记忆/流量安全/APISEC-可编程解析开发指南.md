---
type: product
title: APISEC-可编程解析开发指南
product: APISEC
aliases: []
tags:
    - 流量安全
    - 技术参考
summary: APISEC 可编程解析 Lua 脚本开发指南，用于定制化数据解析与资产发现。
status: verified
---

# APISEC 可编程解析开发指南

## 开发指南

可编程解析的目的是对内置的自动解码

可编程解析会对每一个属于任意应用的站点的请求响应运行，运行实际在流量内置的深度解码之后，API资产匹配之前，即此时是可以获得请求体中的字段值的，并影响该流量可以命中的 API 资产。

### 如何注册

`safeline.match("urlpath contains jsession=", process)`

其中第一个参数是一段匹配逻辑的描述，第二个参数是匹配后将运行的函数。

#### 匹配逻辑

单条匹配逻辑分三个部分 `<field> <operator> <target>`，多条匹配逻辑由 `,` 连接

*   operator 包含
    
    *   equal
        
    *   contains
        
    *   has\_prefix
        
    *   has\_suffix
        
    *   not\_has\_prefix
        
    *   not\_has\_suffix
        
*   同 field 取 or，不同 field 取 and
    
*   field, target 不支持包含空格
    

#### process 函数

`function process(src_ip, host, urlpath)`

函数的三个入参即为触发该插件的消息的 `src_ip`, `host`, `urlpath`。

### Safeline Module

引用方式 `local safeline = require("safeline")`

#### safeline.find\_field(target: string) -> string

在 parsed request 中查找 `key` 为 `target` 的字段，并返回其值。特别的可以使用 urlpath。

未找到时返回 nil。

> 查找少量字段时建议使用该方法减少内存消耗。

#### safeline.find\_multi\_fields(num: int, target: string…) -> (string, string…)

在 parsed request 中查找批量 key 为 target 的字段，并返回其值。特别的可以使用 urlpath。其中 num 为需要查找的 target 数量，后续需要有 num 个 string。返回第一个结果为 err，正常时为 **空字符串**，后续 num 个 string 分别对应查找的 target，未找到的将返回 **空字符串**。

> 查找大量字段时建议使用该方法减少 CPU 消耗。

#### safeline.put\_field(stage: int, key: string, value: string, value\_type: int, value\_origin: int)

向 parsed request 或 parsed response 加入一个解析结果，put\_field 的内容在同一插件中可马上使用 find\_field 获得。

##### 内置常量 - stage

*   `safeline.constants.stage.request`
    
*   `safeline.constants.stage.response`
    

##### 内置常量 - value\_type

*   `safeline.constants.value_type.null`
    
*   `safeline.constants.value_type.bool`
    
*   `safeline.constants.value_type.number`
    
*   `safeline.constants.value_type.string`
    
*   `safeline.constants.value_type.array`
    
*   `safeline.constants.value_type.dict`
    

##### 内置常量 - value\_origin

*   `safeline.constants.value_origin.query`
    
*   `safeline.constants.value_origin.cookie`
    
*   `safeline.constants.value_origin.header`
    
*   `safeline.constants.value_origin.body`
    

#### safeline.log\_info(msg: string)

记录服务底层日志

#### safeline.log\_fatal(msg: string)

记录服务底层日志，但会触发 panic 并阻止插件的后续运行

### Url Module

引用方式 `local url = require("url")`

#### url.escape(source: string) -> string

```plaintext
Encodes a string into its escaped hexadecimal representation
Input
  s: binary string to be encoded

Returns
  escaped representation of string binary

```

#### url.unescape(source: string) -> string

```plaintext
Unencodes a escaped hexadecimal string into its binary representation
Input
  s: escaped hexadecimal string to be unencoded

Returns
  unescaped binary representation of escaped hexadecimal binary

```

#### url.absolute\_path(source: string) -> string

```plaintext
Builds a path from a base path and a relative path
Input
  base_path
  relative_path

Returns
  corresponding absolute path

```

#### url.parse(string) -> table

```plaintext
Parses a url and returns a table with all its parts according to RFC 2396
The following grammar describes the names given to the URL parts
<url> ::= <scheme>://<authority>/<path>;<params>?<query>#<fragment>
<authority> ::= <userinfo>@<host>:<port>
<userinfo> ::= <user>[:<password>]
<path> :: = {<segment>/}<segment>
Input
  url: uniform resource locator of request
  default: table with default values for each field
Returns
  table with the following fields, where RFC naming conventions have
  been preserved:
    scheme, authority, userinfo, user, password, host, port,
    path, params, query, fragment
Obs:
  the leading '/' in {/<path>} is considered part of <path>

```

#### url.build(table) -> string

```plaintext
Rebuilds a parsed URL from its components.
Components are protected if any reserved or unallowed characters are found
Input
  parsed: parsed URL, as returned by parse
Returns
  a stringing with the corresponding URL

```

#### url.parse\_path(source: string) -> array\[string\]

```plaintext
Breaks a path into its segments, unescaping the segments
Input
  path
Returns
  segment: a table with one entry per segment

```

#### url.build\_path(source: array\[string\]) -> string

```plaintext
Builds a path component from its segments, escaping protected characters.
Input
  parsed: path segments
  unsafe: if true, segments are not protected before path is built
Returns
  path: corresponding path stringing

```

### Base64 Module

引用方式 `local base64 = require("base64")`

#### base64.dec(source: string) -> string

base64 解码。

#### base64.enc(source: string) -> string

base64 编码。

### String Moduel

#### safeline.string.startswith(source: string, prefix: string) -> bool

```plaintext
Return True if self starts with the specified prefix, False otherwise.
```

#### safeline.string.endswith(source: string, suffix: string) -> bool

```plaintext
Return True if self ends with the specified suffix, False otherwise.
```

#### safeline.string.lstrip(self: string): string

```plaintext
Return a copy of the string with leading whitespace removed.
```

#### safeline.string.rstrip(self: string): string

```plaintext
Return a copy of the string with trailing whitespace removed.
```

#### safeline.string.strip(self: string): string

```plaintext
Return a copy of the string with leading and trailing whitespace removed.
```

#### safeline.string.replace(self: string, old: string, new: string, \[count: int\]): string

```plaintext
Return a copy with all occurrences of substring old replaced by new.

Optional count argument is maximum number of occurrences to replace,
replace all occurrences by default.
```

#### safeline.string.join(self: string, iter: table): string

```plaintext
Concatenate any number of values in table iter with delimiter self.
Values in table must be string or number.
```

#### safeline.string.split(self: string, delim: string, \[count: int\]): table

```plaintext
Return a list of the words in the string, using delim as the delimiter string.
Raise error when delimiter is empty.

Optional argument count is maximum number of splits to do.
```

## Demo

### jsession in url

```plaintext
local safeline = require("safeline")
local url = require("url")

function process(src_ip, host, urlpath)
    local jsession = ""
    local parts = url.parse_path(urlpath)
    for k, v in ipairs(parts) do
    	local kv = safeline.string.split(v, ";jsession=", 2)
    	if #kv == 2 then
      		jsession = kv[2]
      		parts[k] = kv[1]
        end
    end
    if jsession ~= "" then
        err = safeline.put_field(0, "jsession", jsession, safeline.constants.value_type.string, safeline.constants.value_origin.header)
        if err ~= nil then
            safeline.log_fatal("get err: " .. err)
        end
    	err = safeline.put_urlpath("/" .. safeline.string.join("/", parts))
    	if err ~= nil then
    		safeline.log_fatal("get err: ".. err)  
	    end
    end
    -- safeline.log_info('get jsession: '..safeline.find_field("jsession"))
end

safeline.match("urlpath contains jsession=", process)

```

### action from base64 decoded url

```plaintext
local safeline = require("safeline")
local url = require("url")
local base64 = require("base64")

function process(src_ip, host, urlpath)
    local parts = url.parse_path(urlpath)
    local decoded_action = table.remove(parts)
    local dec_parts = safeline.string.split(decoded_action, '_')
    local result = {}
    for k, v in ipairs(dec_parts) do
        safeline.log_info("get "..k.." value "..v)
        table.insert(result, base64.dec(v))
    end
    safeline.put_field(0, "action", safeline.string.join(".", result),
        safeline.constants.value_type.string,
        safeline.constants.value_origin.header)
    -- safeline.put_urlpath("/" .. safeline.string.join("/", parts))
    -- safeline.log_info('get action: '..safeline.find_field("action"))
end

safeline.match("urlpath contains vat", process)

```

### post data from base64 decoded url

```plaintext
local safeline = require("safeline")
local url = require("url")
local base64 = require("base64")

function process(src_ip, host, urlpath)
    local parts = url.parse_path(urlpath)
    local decoded_data = table.remove(parts)
    local dec_parts = safeline.string.split(decoded_data, '__')
    local action = base64.dec(dec_parts[1])
    local data_raw = base64.dec(dec_parts[2])
    local data = safeline.json.decode(data_raw)
    safeline.put_field(0, "action", action,
        safeline.constants.value_type.string,
        safeline.constants.value_origin.body)
    for k, v in pairs(data) do
        safeline.put_field(0, k, v,
            safeline.constants.value_type.string,
            safeline.constants.value_origin.body)    
    end

    safeline.put_urlpath("/" .. safeline.string.join("/", parts))
    -- safeline.log_info('get action: '..safeline.find_field("action"))
end

safeline.match("urlpath contains gw", process)
```
