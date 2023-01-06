ARG branch=latest
FROM gcr.io/evmchain/internal-devnet-protocol:$branch

ADD start.sh /start.sh
RUN chmod +x /start.sh
RUN npm pkg get version >/version.txt

RUN mkdir /hive-bin
ADD deploy.sh /hive-bin/deploy.sh
RUN chmod +x /hive-bin/deploy.sh

ENTRYPOINT ["/start.sh"]
