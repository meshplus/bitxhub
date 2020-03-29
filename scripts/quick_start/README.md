# Quick start

## Fabric first network

Download script:

```bash
wget https://github.com/meshplus/bitxhub/raw/master/scripts/quick_start/ffn.sh
```

```bash
./ffn.sh up // build up fabric first network
./ffn.sh down // clear up fabric first network
./ffn.sh restart // restart fabric first network
```

`./ffn.sh up` will generate `crypto-config`.

## Chaincode

Download script:

```bash
wget https://github.com/meshplus/bitxhub/raw/master/scripts/quick_start/chaincode.sh
```

```bash
./chaincode.sh install <fabric_ip> // Install broker, transfer and data_swapper chaincode
./chaincode.sh upgrade <fabric_ip> <chaincode_version> 
./chaincode.sh init <fabric_ip> // Init broker chaincode
./chaincode.sh get_balance <fabric_ip> // Query Alice balance
./chaincode.sh get_data <fabric_ip> // Query the value of 'path'
./chaincode.sh interchain_transfer <fabric_ip> <target_appchain_id>
./chaincode.sh interchain_gt <fabric_ip> <target_appchain_id>
```

## Fabric pier
Download script:

```bash
wget https://github.com/meshplus/bitxhub/raw/master/scripts/quick_start/fabric_pir.sh
```

```bash
./fabric_pier.sh start  <bitxhub_addr> <fabric_ip> <pprof_port> 
./fabric_pier.sh restart  <bitxhub_addr> <fabric_ip> <pprof_port> 
./fabric_pier.sh id
```

