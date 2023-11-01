# rpc-proxy

### How to Use

```sh
> rpc-proxy help
NAME:
   rpc-proxy - A proxy for web3 JSONRPC

USAGE:
   rpc-proxy [global options] command [command options] [arguments...]

VERSION:
   0.0.60

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --allowedfunc value  comma separated list of allowed paths
   --blocklimit value   block range query limit (default: 0)
   --chainId value      chainId (default: 12345)
   --config value       path to toml config file
   --help, -h           show help (default: false)
   --nolimit value      list of ips allowed unlimited requests(separated by commas)
   --port value         port to export to (default: 8540)
   --rpm value          limit for number of requests per minute from single IP (default: 1000)
   --url value          local chain http url (default: "http://127.0.0.1:8545")
   --version, -v        print the version (default: false)
   --wsurl value        local chain websocket url (default: "ws://127.0.0.1:8546")
```

## 应用
1. 以太坊Node 对外暴露 8540 接口（示例）。
2. 将 8540 收到的数据包经过解析后转到8545端口
3. 解析逻辑-只允许特定地址部署合约
```text
如果TO==nil (合约部署交易)
判断sender是否是我们授权的节点/本地节点
if true{
    转发包}
 else{
    return 不允许别的地址部署链上合约
 }
```

4. 工具用法：
对外公开 8540 端口，转到本地的8545端口

config内部定义节点支持的ETH功能

./rpc-proxy --config ./config.toml

config.toml内部定义支持部署合约的账户地址和允许的链上操作：

# 示例
```toml
# This is an example configuration file.
# 区块链开放的功能
Allow = [
  "eth_blockNumber", 
  "eth_chainId",
  "eth_estimateGas",
  "eth_gasPrice",
  "eth_getBalance",
  "eth_getBlockByHash",
  "eth_getBlockByNumber",
  "eth_getBlockTransactionCountByHash",
  "eth_getBlockTransactionCountByNumber",
  "eth_getCode",
  "eth_getLogs",
  "eth_getStorageAt",
  "eth_getTransactionByBlockHashAndIndex",
  "eth_getTransactionByBlockNumberAndIndex",
  "eth_getTransactionByHash",
  "eth_getTransactionCount",
  "eth_getTransactionReceipt",
  "eth_newBlockFilter",
  "eth_newPendingTransactionFilter",
  "eth_sendRawTransaction",
  "eth_subscribe",
  "eth_unsubscribe",
  "net_listening",
  "net_version", 
]

# 允许部署合约的Sender地址
SCAddress = [
  "0x60a6e5af0525523a617xxxcf6c1f85353fba0408a7b",
  "xx",
]

# 限制 IP 每分钟发送的RPC请求数
RPM = 1000

# 白名单IP，IP地址不受RPC请求数的限制
NoLimit = [] # ["192.168.101.75", "192.168.101.75"]

# 在订阅区块链上事件时，允许订阅的最大区块跨度，跨度越大，就需要消耗越多的计算、存储资源
BlockRangeLimit = 0

# 对外开放的端口
Port = 8540

# 本地链的HTTP端口
URL = "http://127.0.0.1:8545" #RPC URL

# 本地链的WebSorket端口
WSURL = "ws://127.0.0.1:8546" #RPC URL

ChainID = 12345
```

5. 主要代码片段：
 解析数据包
 ```golang
 func parseRequests(r *http.Request) (string, []string, []ModifiedRequest, error) {
    if methods== "eth_sendRawTransaction" {
        rawData :=Params
        bytes, _ := hexutil.Decode(strings.Trim(string(rawData), `"`))
        tx := new(types.Transaction)
        if err := tx.UnmarshalBinary(bytes); err != nil {
            return "", nil, nil, err
        }
        toAddr := tx.To()
        senderAddr := strings.ToLower(sender.Hex())
        if toAddr != nil { //普通转账交易
            fmt.Println(fmt.Sprintf("TRANSFER_FROM_%s_TO_%s", sender.Hex(), toAddr.Hex()))
        } else {
            if !SCAddress[senderAddr] { //如果不是配置中的地址，则不允许部署合约
                return "", nil, nil, fmt.Errorf("NOT_APPROVED_DEPLOY_CONTRACT")
            }
            fmt.Println(fmt.Sprintf("DEPLOY_CONTRACT_FROM_%s", senderAddr))
        }
    }
    return ip, methods, res, nil
}
 ```
