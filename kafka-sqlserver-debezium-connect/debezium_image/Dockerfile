FROM debezium/connect:2.3

COPY entrypoint.sh /debezium/entrypoint.sh
COPY configure-connector.sh /debezium/configure-connector.sh

ENTRYPOINT ["/debezium/entrypoint.sh"]
