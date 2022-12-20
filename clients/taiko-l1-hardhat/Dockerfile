ARG branch=latest
FROM gcr.io/evmchain/hardhat-node:$branch

# Inject the enode id retriever script
RUN mkdir /hive-bin
ADD enode.sh /hive-bin/enode.sh
RUN chmod +x /hive-bin/enode.sh

COPY ./start.sh /start.sh
RUN chmod +x /start.sh

RUN echo "20220101" >/version.txt

ENTRYPOINT ["/start.sh"]
