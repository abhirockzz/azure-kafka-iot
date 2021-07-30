package com.abhirockzz.deviceapi;

import com.azure.spring.data.cosmos.core.mapping.Container;
import com.azure.spring.data.cosmos.core.mapping.PartitionKey;
import org.springframework.data.annotation.Id;

@Container(containerName = "avg_temp_enriched", autoCreateContainer = false)
public class DeviceStats {
    @Id
    @PartitionKey
    private String id;
    private Double avg_temp;
    private String info;

    public DeviceStats() {

    }

    public DeviceStats(String id, Double avg_temp, String info) {
        this.id = id;
        this.avg_temp = avg_temp;
        this.info = info;
    }

    public String getId() {
        return id;
    }

    public void setId(String id) {
        this.id = id;
    }

    public Double getAvg_temp() {
        return avg_temp;
    }

    public void setAvg_temp(Double avg_temp) {
        this.avg_temp = avg_temp;
    }

    public String getInfo() {
        return info;
    }

    public void setInfo(String info) {
        this.info = info;
    }

}