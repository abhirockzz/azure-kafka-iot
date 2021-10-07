# Extracting Value from IOT using Azure Cosmos DB, Azure Synapse Analytics and Confluent Cloud

Our (hypothetical) requirement is to deal ingest, store and analyse device data such as temperature and pressure) in different locations. Mock device data is generated using a program. This data flows into a MQTT broker and the Kafka [MQTT source connector](https://docs.confluent.io/kafka-connect-mqtt/current/mqtt-source-connector/index.html) is used to it to a Confluent Cloud Kafka cluster on Azure. From there on, the [Kafka Connect Sink connector for Azure Cosmos DB](https://docs.microsoft.com/azure/cosmos-db/kafka-connector-sink?WT.mc_id=data-28802-abhishgu) takes care of sending the device data from Confluent Cloud to Azure Cosmos DB.

Both the connectors run in Azure Kubernetes Service as separate [Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/)s. Once the pipeline is operational and the device data is flowing into Azure Cosmos DB, it should be put to good use! This tutorial covers a couple of examples:

- One of them is how to [Azure Synapse Link](https://docs.microsoft.com/azure/cosmos-db/synapse-link?WT.mc_id=data-28802-abhishgu) to process data in Azure Cosmos DB using [Apache Spark pools](https://docs.microsoft.com/azure/synapse-analytics/spark/apache-spark-overview?WT.mc_id=data-28802-abhishgu) in [Azure Synapse Analytics](https://docs.microsoft.com/azure/synapse-analytics/overview-what-is?WT.mc_id=data-28802-abhishgu). The goal is to aggregate data, create [materialized view](https://en.wikipedia.org/wiki/Materialized_view) and make it readily available to other services.
- One such service is a Spring Boot Java application running on [Azure Spring Cloud](https://docs.microsoft.com/azure/spring-cloud/?WT.mc_id=data-28802-abhishgu) built using the [Azure Cosmos DB Spring support](https://docs.microsoft.com/azure/developer/java/spring-framework/how-to-guides-spring-data-cosmosdb?WT.mc_id=data-28802-abhishgu), which exposes REST endpoints to access data in Azure Cosmos DB.

## Pre-requisites

- An Azure account - [you can get one for free here](https://azure.microsoft.com/free/?WT.mc_id=data-28802-abhishgu)
- Install [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli?WT.mc_id=data-28802-abhishgu)
- JDK 11 for e.g. [OpenJDK](https://openjdk.java.net/projects/jdk/11/])
- A recent version of [Maven](https://maven.apache.org/download.cgi) and [Git](https://git-scm.com/downloads)
- Docker

## Infrastructure setup

Provision [Confluent Cloud cluster on Azure Marketplace](https://docs.microsoft.com/azure/partner-solutions/apache-kafka-confluent-cloud/create?WT.mc_id=data-28802-abhishgu). Create a new Dedicated cluster in Confluent Cloud with Private Link enabled, set up a private endpoint in your Azure VNet, and securely connect to Confluent Cloud Kafka cluster for data in motion from your Azure VNet. Configure Azure Private Link for Dedicated clusters in Azure [using these instructions](https://docs.confluent.io/cloud/current/networking/azure-privatelink.html#cloud-networking-privatelink-azure).

Once that's done, create the following:

- [Confluent Cloud API Key and Secret](https://docs.confluent.io/cloud/current/client-apps/api-keys.html#create-resource-specific-api-keys-in-the-ui): The MQTT and Azure Cosmos DB connectors need credentials to access the Kafka cluster.

- A Kafka topic named `mqtt.device-stats`. When using Private Link, the Confluent Cloud UI componentslike topic management etc. are not publicly reachable. You must [configure internal access to these components](https://docs.confluent.io/cloud/current/networking/peering-ui-access.html)

To setup Azure Cosmos DB, start by [creating an account using the Azure portal](https://docs.microsoft.com/en-us/azure/cosmos-db/create-cosmosdb-resources-portal#create-an-azure-cosmos-db-account). Turn on the Azure Synapse Link feature (navigate to the `Features` section in the Azure portal).

Create a database (named `iotdb`) and containers with the below mentioned specifications:

- `device-data`: Partition key - `/location`, Analytical store `On`
- `avg_temp`: Partition key - `/id`, Analytical store `Off`
- `locations`: Partition key - `/id`, Analytical store `On`
- `avg_temp_enriched`: Partition key - `/id`, Analytical store `Off`

> All the containers should show up under the `iotdb` database

[Provision a Azure Synapse Workspace using the portal](https://docs.microsoft.com/azure/synapse-analytics/quickstart-create-workspace?WT.mc_id=data-28802-abhishgu#create-a-synapse-workspace). Once that's done, you can [follow these instructions](https://docs.microsoft.com/azure/synapse-analytics/quickstart-create-apache-spark-pool-studio?WT.mc_id=data-28802-abhishgu) to create a serverless Apache Spark pool. Finally, create a Linked Service for Azure Cosmos DB.

Install a Kubernetes cluster on Azure [using the portal](https://docs.microsoft.com/en-us/azure/aks/kubernetes-walkthrough-portal#create-an-aks-cluster) or [Azure CLI](https://docs.microsoft.com/en-us/azure/aks/kubernetes-walkthrough#create-aks-cluster) - make sure it's in the same VNet as per Private Link config.

In order to connect to the cluster from your local machine using `kubectl`, use the below command:

```
az aks get-credentials --resource-group <AZURE_RESOURCE_GROUP> --name <AKS_CLUSTER_NAME>
```

To confirm:

```bash
kubectl get nodes

#output will be similar to
NAME                                STATUS   ROLES   AGE   VERSION
aks-agentpool-12345678-vmss000000   Ready    agent   1d    v1.20.7
```

Create an instance of Azure Spring Cloud using the Azure portal [using this guide](https://docs.microsoft.com/en-us/azure/spring-cloud/quickstart?tabs=Azure-CLI&pivots=programming-language-java#provision-an-instance-of-azure-spring-cloud).

Clone this GitHub repo before you proceed further:

```bash
git clone https://github.com/abhirockzz/azure-kafka-iot
cd azure-kafka-iot
```

## Send device data to Kafka using MQTT source connector

This tutorial makes use of a [publicly hosted HiveMQ MQTT broker](https://www.hivemq.com/public-mqtt-broker/). This will make things quite simple since you don't have to worry about setting up a broker yourself.

> The public `HivemQ` broker is available at `broker.hivemq.com:1883` and it is open for anyone. This tutorial makes use of it for experimental/instructional purposes only.

Update the `Secret` manifest file `kafka-cluster-secret.yaml` (under `mqtt-source-connector/deploy`) with API Key and Secret for Confluent Cloud. 

To deploy the connector:

```bash
kubectl apply -f mqtt-source-connector/deploy/

#expected output
secret/kafka-cluster-credentials created
configmap/mqtt-connector-config created
deployment.apps/mqtt-kafka-connector created
```

**Create MQTT source connector instance**

Make sure to update the `mqtt-source-config.json` (under `mqtt-source-connector`) and enter the correct values for `confluent.topic.bootstrap.servers` and `confluent.topic.sasl.jaas.config`. Here is the connector configuration snippet:

```json
{
    "name": "mqtt-source",
    "config": {
        "connector.class": "io.confluent.connect.mqtt.MqttSourceConnector",
        "tasks.max": "1",
        "mqtt.server.uri": "tcp://broker.hivemq.com:1883",
        "mqtt.topics": "device-stats",
        "kafka.topic": "mqtt.device-stats",
        ...
        "confluent.topic.replication.factor": 3,
        "confluent.topic.bootstrap.servers": "<confluent cloud bootstrap server>",
        "confluent.topic.security.protocol": "SASL_SSL",
        "confluent.topic.sasl.jaas.config": "org.apache.kafka.common.security.plain.PlainLoginModule required username='<confluent cloud API key>' password='confluent cloud API secret';",
```

> The MQTT topic named `device-stats` is mapped to `mqtt.device-stats` topic in Kafka. Leave these values unchanged

To keep things simple, we can use port forwarding to access the REST interface for the Kafka Connect instance in Kubernetes:

```bash
kubectl port-forward $(kubectl get pods -l=app=mqtt-kafka-connector --output=jsonpath={.items..metadata.name}) 8083:8083
```

Use the Kafka Connect REST interface to create the connector instance:

```bash
curl -X POST -H 'Content-Type: application/json' http://localhost:8083/connectors -d @mqtt-source-connector/mqtt-source-config.json

# wait for a minute before checking the connector status
curl http://localhost:8083/connectors/mqtt-source/status
```

**Publish mock data**

An application to generate device data has been packaged as a [pre-built Docker image](https://hub.docker.com/repository/docker/abhirockzz/mqtt-go-producer) for your convenience. It will start publishing mock JSON data to an MQTT broker endpoint:

> [Here is the Dockerfile](mqtt-publisher/Dockerfile) for this application.

To start the producer application:

```bash
docker run -e MQTT_BROKER=tcp://broker.hivemq.com:1883 abhirockzz/mqtt-go-producer
```

If everything works well, you should see data flowing into Confluent Cloud. Check the `mqtt.device-stats` topic.

Now that our source setup is operational, its time to focus on the part that will persist the device data in the Kafka topic to a container in Azure Cosmos DB using the [Kafka Connect Sink connector for Azure Cosmos DB](https://docs.microsoft.com/azure/cosmos-db/kafka-connector-sink?WT.mc_id=data-28802-abhishgu).

## Send device data from Confluent Cloud topic to Azure Cosmos DB

To deploy the Azure Cosmos DB connector:

```bash
kubectl apply -f cosmosdb-sink-connector/deploy/

#expected output
configmap/connector-config created
deployment.apps/cosmosdb-kafka-connector created
```

Wait for the Kafka Connect instance to start - it might take a minute or so. In the meantime, you can keep track of the logs:

```bash
kubectl logs -f $(kubectl get pods -l=app=cosmosdb-kafka-connector --output=jsonpath={.items..metadata.name})
```

Make sure to update the connector configuration with the Azure Cosmos DB account name (in `connect.cosmos.connection.endpoint`) and the Access Key (for `connect.cosmos.master.key` attribute). Leave the others unchanged.

Just like before, we can use port forwarding To access the REST interface for the Kafka Connect instance:

```bash
kubectl port-forward $(kubectl get pods -l=app=cosmosdb-kafka-connector --output=jsonpath={.items..metadata.name}) 9090:8083
```

In another terminal, enter the below command to start the Azure Cosmos DB connector instance:

```bash
curl -X POST -H "Content-Type: application/json" -d @cosmosdb-sink-connector/connector-config.json http://localhost:9090/connectors

# wait for few seconds before checking the status and making sure its RUNNING
curl http://localhost:9090/connectors/iot-sink/status
```

Since the device records are already flowing into Kafka via MQTT source connector (in the `mqtt.device-stats` topic), it should get persisted to Azure Cosmos DB. Navigate to the Azure portal and check the `device-data` container:

We have the pipeline setup, good progress so far! It's time to process this data.

## Near real-time processing using Azure Synapse Analytics

> The rest of the steps are a part of [this notebook](synapse/iot-analytics.ipynb).

We start by reading data from the `device-data` container in Azure Comsmos DB. Note that this is executed against the [analytical store](https://docs.microsoft.com/azure/cosmos-db/analytical-store-introduction?WT.mc_id=data-28802-abhishgu) (notice the `cosmos.olap` format) which is a fully isolated column store to run analytics without affecting your transactional workloads.

```python
iotdata = spark.read.format("cosmos.olap")\
            .option("spark.synapse.linkedService", "iotcosmos")\
            .option("spark.cosmos.container", "device-data")\
            .load()

print(iotdata.count())
display(iotdata.limit(3))
```

This is followed by a basic aggregation which involves calculating the average temperature across all devices in each location:

```python
average_temp = iotdata.groupBy("location").avg("temp") \
                        .withColumnRenamed("avg(temp)", "avg_temp") \
                        .orderBy("avg(temp)")

display(average_temp)
```

This `DataFrame` is written to another Azure Cosmos DB container (`avg_temp`).

> The `id` attribute is used as the partition key (which is populated with the same value as the location name). This allows you to do point queries (least expensive) to get the average temperature across all devices for a specific location.

Writing to Azure Cosmos DB is simple as well. Please note that writes always go to the OLTP container (note `cosmos.oltp` format) and consume Request Units provisioned on the Azure Cosmos DB container:

```python
average_temp_new = average_temp \
                    .withColumn("id", average_temp["location"]) \
                    .drop("location")

average_temp_new.write\
    .format("cosmos.oltp")\
    .option("spark.synapse.linkedService", "iotcosmos")\
    .option("spark.cosmos.container", "avg_temp")\
    .option("spark.cosmos.write.upsertEnabled", "true")\
    .mode('append')\
    .save()
```

Navigate to the Azure portal and check the `avg_temp` container in Azure Cosmos DB.

Note that the location data is of the form `location-1`, `location-2` etc. It does not make a lot of sense. What if we enrich this with the actual location name? It's a good excuse to learn how to use Spark SQL tables in Synapse and join it with data from Azure Cosmos DB!

First, we need the location metadata to be stored in Azure Cosmos DB. [It's a CSV file](location-metadata.csv) whose contents are:

```
name,info
location-1,New Delhi
location-2,New Jersey
location-3,New Orleans
location-4,Seattle
location-5,Colorado
location-6,Toronto
location-7,Montreal
location-8,Ottawa
location-9,New york
```

Upload this file to the ADLS filesystem associated with the Azure Synapse workspace.

Read the `CSV` info into a Data Frame:

```python
locationInfo = (spark
                .read
                .csv("/location-metadata.csv", header=True, inferSchema='true')
              )

display(locationInfo)
```

... and write that data to Azure Cosmos DB:

```python
locations = locationInfo \
                    .withColumn("id", locationInfo["name"]) \
                    .drop("name")

locations.write\
            .format("cosmos.oltp")\
            .option("spark.synapse.linkedService", "iotcosmos")\
            .option("spark.cosmos.container", "locations")\
            .option("spark.cosmos.write.upsertEnabled", "true")\
            .mode('append')\
            .save()
```

Navigate to the Azure portal and check the `locations` container:

Start by creating a Spark database:

```sql
%%sql
create database iotcosmos
```

Create Spark tables on top of raw device data and the location metadata in Azure Cosmos DB. First one is `iotcosmos.iot_data`:


```sql
%%sql
create table if not exists iotcosmos.iot_data using cosmos.olap options (
    spark.synapse.linkedService 'iotcosmos',
    spark.cosmos.container 'device-data'
)
```

The second table is `iotcosmos.locations`:

```sql
%%sql

create table if not exists iotcosmos.locations using cosmos.olap options (
    spark.synapse.linkedService 'iotcosmos',
    spark.cosmos.container 'locations'
)
```

`JOIN` the data in different tables, based on `location`:

```python
avg_temp_enriched = spark.sql("select b.info, \
                            a.location, \
                            AVG(a.temp) \
                            from iotcosmos.iot_data a \
                             join iotcosmos.locations b \
                            on a.location = b.id \
                            group by a.location, b.info")
                    

display(avg_temp_enriched)
```

The result is the average temperature across all devices in a location with the location name (enriched information). Finally, we write it to the `avg_temp_enriched` container in Azure Cosmos DB.

```python
avg_temp_enriched_with_id = avg_temp_enriched \
                    .withColumn("id", avg_temp_enriched["location"]) \
                    .withColumnRenamed("avg(temp)", "avg_temp") \
                    .drop("location")

display(avg_temp_enriched_with_id)

avg_temp_enriched_with_id.write\
    .format("cosmos.oltp")\
    .option("spark.synapse.linkedService", "iotcosmos")\
    .option("spark.cosmos.container", "avg_temp_enriched")\
    .option("spark.cosmos.write.upsertEnabled", "true")\
    .mode('append')\
    .save()
```

Navigate to the Azure portal and check the `avg_temp_enriched` container.

In the final section, you will deploy a Java app on Azure Spring Cloud to expose the aggregated temperature data in Azure Cosmos DB.

## Access Spring Boot application to check device data

Start by creating the Azure Spring Cloud application using Azure CLI. First, install the Azure Spring Cloud [extension for the Azure CLI](https://docs.microsoft.com/cli/azure/extension?view=azure-cli-latest&amp;WT.mc_id=data-28802-abhishgu):

```bash
az extension add --name spring-cloud
```

[Create the Azure Spring Cloud applications](https://docs.microsoft.com/cli/azure/ext/spring-cloud/spring-cloud/app?view=azure-cli-latest&amp;WT.mc_id=data-28802-abhishgu#ext_spring_cloud_az_spring_cloud_app_create):

``` bash
az spring-cloud app create -n device-stats -s <enter the name of Azure Spring Cloud service instance> -g <enter azure resource group name> --runtime-version Java_11 --assign-endpoint true
```

> `device-stats` is the application name

Before deploying the application, update the `application.properties` (in `device-data-api/src/main/resources`) file with Azure Cosmos DB account name and Access key:

```properties
azure.cosmos.uri=https://<account name>.documents.azure.com:443/
azure.cosmos.key=<access key>
azure.cosmos.database=iotdb
cosmos.queryMetricsEnabled=true
```

Build the application JAR file:

```bash
cd device-data-api
export JAVA_HOME=<enter path to JDK e.g. /Library/Java/JavaVirtualMachines/zulu-11.jdk/Contents/Home>
mvn clean package
```

To [deploy the application JAR file](https://docs.microsoft.com/cli/azure/ext/spring-cloud/spring-cloud/app?view=azure-cli-latest&amp;WT.mc_id=data-28802-abhishgu#ext_spring_cloud_az_spring_cloud_app_deploy):

```bash
az spring-cloud app deploy -n device-stats -s <enter the name of Azure Spring Cloud service instance> -g <enter azure resource group name> --jar-path target/device-data-api-0.0.1-SNAPSHOT.jar
```

While the application gets deployed and starts up, you can check the logs to monitor its progress:

```bash
az spring-cloud app logs -n device-stats -s <enter the name of Azure Spring Cloud service instance> -g <enter azure resource group name>
```

Navigate to the Azure portal and confirm that the application is up and running.

To invoke the REST API, we need to find the endpoint at which our application is accessible. You can just use the Azure portal for that, but its also possible to do it via Azure CLI:

```bash
az spring-cloud app show -n device-stats -s <enter the name of Azure Spring Cloud service instance> -g <enter azure resource group name> --query 'properties.url'
```

There are two endpoints supported by the app:

- One for listing all the devices with their respective (average) temperature readings
- The other for checking a specific device using its `id` (which in this example, happens to be location-1, location-2 etc.)

> You can invoke the APIs with your browser or use a CLI tool such as `curl`

To find info for all devices:

> where `az-spring-cloud` is the name of the Azure Spring Cloud service instance

```bash
curl https://az-spring-cloud-device-stats.azuremicroservices.io/devices

# JSON output
[
  {
    "id": "location-6",
    "avg_temp": 29.21153846153846,
    "info": "Toronto"
  },
  {
    "id": "location-5",
    "avg_temp": 25.482142857142858,
    "info": "Colorado"
  },
  {
    "id": "location-2",
    "avg_temp": 23.28,
    "info": "New Jersey"
  },
  {
    "id": "location-4",
    "avg_temp": 21.6875,
    "info": "Seattle"
  },
  {
    "id": "location-8",
    "avg_temp": 22.408163265306122,
    "info": "Ottawa"
  },
  {
    "id": "location-3",
    "avg_temp": 27.428571428571427,
    "info": "New Orleans"
  },
  {
    "id": "location-7",
    "avg_temp": 27.82857142857143,
    "info": "Montreal"
  },
  {
    "id": "location-9",
    "avg_temp": 21.9811320754717,
    "info": "New york"
  },
  {
    "id": "location-1",
    "avg_temp": 25.53846153846154,
    "info": "New Delhi"
  }
]
```

For a specific location (`location-2`):

> where `az-spring-cloud` is the name of the Azure Spring Cloud service instance

```bash
curl https://az-spring-cloud-device-stats.azuremicroservices.io/devices/location-2

# JSON output
{
  "id": "location-2",
  "avg_temp": 25.337579617834393,
  "info": "New Jersey"
}
```
