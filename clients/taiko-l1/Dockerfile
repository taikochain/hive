FROM ethereum/client-go:latest

RUN apk add --update bash curl jq

# Inject the enode id retriever script
RUN mkdir /hive-bin
ADD enode.sh /hive-bin/enode.sh
RUN chmod +x /hive-bin/enode.sh

COPY ./genesis.json /host/genesis.json
COPY ./start.sh /start.sh
RUN chmod +x /start.sh

RUN /usr/local/bin/geth console --exec 'console.log(admin.nodeInfo.name)' --maxpeers=0 --nodiscover --dev 2>/dev/null | head -1 >/version.txt

ENTRYPOINT ["/start.sh"]
