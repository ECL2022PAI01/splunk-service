package main

import splunkalert "github.com/kuro-jojo/splunk-sdk-go/src/alerts"

const UnhandleKeptnCloudEvent = "Unhandled Keptn Cloud Event : "

// ServiceName specifies the current services name (e.g., used as source when sending CloudEvents)
const ServiceName = "splunk-service"

// These variables facilitate tests
var cloudEventListener = CloudEventListener
var processKeptnCloudEvent = ProcessKeptnCloudEvent
var handleConfigureMonitoringTriggeredEvent = HandleConfigureMonitoringTriggeredEvent
var handleGetSliTriggeredEvent = HandleGetSliTriggeredEvent
var createAlert = splunkalert.CreateAlert
