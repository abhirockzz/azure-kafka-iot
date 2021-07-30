package com.abhirockzz.deviceapi;

import java.util.List;

import com.azure.spring.data.cosmos.repository.CosmosRepository;
import com.azure.spring.data.cosmos.repository.Query;

import org.springframework.stereotype.Repository;

@Repository
public interface DeviceStatsRepository extends CosmosRepository<DeviceStats, String> {

    @Query(value = "SELECT * FROM c")
    List<DeviceStats> getAllDeviceStats();
}
