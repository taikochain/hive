ARG branch=main
FROM gcr.io/evmchain/taiko-client:$branch

ADD start.sh /start.sh
RUN chmod +x /start.sh
RUN taiko-client -v > /version.txt

ENTRYPOINT ["/start.sh"]