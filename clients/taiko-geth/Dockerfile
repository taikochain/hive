ARG branch=taiko
FROM gcr.io/evmchain/taiko-geth:$branch

ADD start.sh /start.sh
RUN /usr/local/bin/geth console --exec 'console.log(admin.nodeInfo.name)' --maxpeers=0 --nodiscover --dev 2>/dev/null | head -1 > /version.txt

ENTRYPOINT ["/start.sh"]