FROM confluentinc/cp-kafka-connect-base:latest
RUN confluent-hub install --no-prompt microsoftcorporation/kafka-connect-cosmos:1.0.4-beta
COPY InsertUuid-1.0-SNAPSHOT.jar /usr/share/confluent-hub-components/