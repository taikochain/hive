FROM gcr.io/evmchain/protocol:latest

ADD start.sh /start.sh
RUN chmod +x /start.sh
RUN npm pkg get version > /version.txt

ENTRYPOINT ["/start.sh"]

