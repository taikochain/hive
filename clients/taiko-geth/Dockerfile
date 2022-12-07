ARG branch=taiko
FROM gcr.io/evmchain/taiko-geth:$branch

RUN apk add --update bash curl jq

RUN /usr/local/bin/geth console --exec 'console.log(admin.nodeInfo.name)' --maxpeers=0 --nodiscover --dev 2>/dev/null | head -1 >/version.txt

# Inject the startup script
ADD geth.sh /geth.sh
RUN chmod +x /geth.sh

# Inject the enode id retriever script
RUN mkdir /hive-bin
ADD enode.sh /hive-bin/enode.sh
RUN chmod +x /hive-bin/enode.sh

# ADD start.sh /start.sh
# RUN chmod +x /start.sh

ENTRYPOINT ["/geth.sh"]
