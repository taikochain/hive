ARG branch=taiko
FROM gcr.io/evmchain/taiko-geth:$branch

RUN apk add --update bash curl jq

RUN /usr/local/bin/geth console --exec 'console.log(admin.nodeInfo.name)' --maxpeers=0 --nodiscover --dev 2>/dev/null | head -1 >/version.txt

# Inject the startup script
COPY geth.sh /geth.sh
RUN chmod +x /geth.sh

# Inject the enode id retriever script
RUN mkdir /hive-bin

COPY enode.sh /hive-bin/enode.sh
RUN chmod +x /hive-bin/enode.sh

COPY deploy_result.sh /hive-bin/deploy_result.sh
RUN chmod +x /hive-bin/deploy_result.sh

ENTRYPOINT ["/geth.sh"]
