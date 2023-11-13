# rpc-proxy

## 应用

1. 以太坊Node 对外暴露 EXPORT_PORT 接口（示例）。
2. 将 EXPORT_PORT 收到的数据包经过解析后转到 ETHEREUM_HTTP_URL 端口
3. 解析逻辑-只允许 ALLOW_CONTRACTS_DEPLOYER 特定地址部署合约
4. .env文件中声明环境变量 

```text
 ALLOW_CMDS // 对外开放的链功能
 ALLOW_CONTRACTS_DEPLOYER //允许部署合约的地址
 EXPORT_PORT //对外公开的端口
 ETHEREUM_HTTP_URL //转发到的以太坊 HTTP 端口
 ETHETEUM_WS_URL //转发到的以太坊 WS 端口
 CHAIN_ID //以太坊 链ID 
```

举例：

```text
export ALLOW_CMDS=eth_blockNumber,eth_call,eth_chainId,eth_estimateGas,eth_gasPrice,eth_getBalance,eth_getBlockByHash,eth_getBlockByNumber,eth_getBlockTransactionCountByHash,eth_getBlockTransactionCountByNumber,eth_getTransactionByHash,eth_getTransactionCount,eth_getTransactionReceipt,eth_sendRawTransaction,net_listening,net_version
export ALLOW_CONTRACTS_DEPLOYER=0x60a6e5af0525523a617cf6c1f85353fba0408a7b,0x68d866baafa993bc002cd35218c13f10ac54221d,0xdd15a18b453eb92140a149f774d1c792919bb352
export EXPORT_PORT=3000
export ETHEREUM_HTTP_URL=http://127.0.0.1:8545
export ETHETEUM_WS_URL=ws://127.0.0.1:8545
export CHAIN_ID=515
```

### How to Use
1. 直接编译使用

```go build && ./rpc-proxy```

2. Docker 容器

```docker build -t gochain/rpc-proxy .```

```docker compose up -d```

docker-compose.yaml 示例

```yaml
version: '3.8'

services:
  chain-rpc:
    image: gochain/rpc-proxy:latest
    ports:  
      - 3000:3000
    container_name: 'chain-rpc'
    volumes:
      - ./.env:/app/.env   
```

查询区块测试：

```curl -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' -H "Content-Type: application/json" http://localhost:3000```