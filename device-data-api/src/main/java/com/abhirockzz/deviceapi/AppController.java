package com.abhirockzz.deviceapi;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@Controller
@RequestMapping(path = "/devices")
public class AppController {

    @Autowired
    private DeviceStatsRepository statsRepo;

    public AppController() throws Exception {
    }

    @GetMapping("/{id}")
    public @ResponseBody DeviceStats getDeviceStatsByLocation(@PathVariable("id") String locationID) {
        System.out.println("searching for device with id " + locationID);

        return statsRepo.findById(locationID).get();
    }

    @GetMapping("")
    public @ResponseBody List<DeviceStats> getAllDeviceStats() {
        System.out.println("searching all devices");
        return statsRepo.getAllDeviceStats();
    }
}
