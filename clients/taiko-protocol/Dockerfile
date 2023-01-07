ARG branch=latest
FROM gcr.io/evmchain/internal-devnet-protocol:$branch

COPY start.sh /start.sh
RUN chmod +x /start.sh
RUN npm pkg get version >/version.txt

RUN mkdir /hive-bin
COPY deploy.sh /hive-bin/deploy.sh
RUN chmod +x /hive-bin/deploy.sh

# change some config for test
COPY LibSharedConfig.sol /taiko-mono/packages/protocol/contracts/libs/LibSharedConfig.sol

ENTRYPOINT ["/start.sh"]
