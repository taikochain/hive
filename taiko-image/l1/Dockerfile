FROM ethereum/client-go:latest

RUN apk add --update bash curl jq

# Init geth config
COPY genesis.json /host/genesis.json
COPY private_key /host/private_key
COPY private_key_pwd.txt /host/private_key_pwd.txt

RUN geth init --datadir /data/l1-node /host/genesis.json
RUN geth account import --password /host/private_key_pwd.txt --datadir /data/l1-node /host/private_key

# Inject the enode id retriever script
RUN mkdir /hive-bin
ADD enode.sh /hive-bin/enode.sh
RUN chmod +x /hive-bin/enode.sh

COPY deploy_result.sh /hive-bin/deploy_result.sh
RUN chmod +x /hive-bin/deploy_result.sh

COPY start.sh /start.sh
RUN chmod +x /start.sh

ENTRYPOINT ["/start.sh"]
