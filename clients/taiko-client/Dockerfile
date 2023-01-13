FROM taiko-client:local

ADD start.sh /start.sh
RUN chmod +x /start.sh
RUN taiko-client -v >/version.txt

ENTRYPOINT ["/start.sh"]
