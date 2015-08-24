var CACHE_SIZE = 25
var MAX_LABELS = 20
var MAX_BINS = 100

var chart = null
var cache = {}
var cacheMRU = []

function cacheUpdateMRU(id) {
    function indexInCache(id) {
        var i;
        for (i = 0; i < cacheMRU.length; i++) {
            if (cacheMRU[i] === id) {
                return i
            }
        }

        return -1
    }

    var i = indexInCache(id)
    if (i >= 0) {
        // Is in cache
        cacheMRU.splice(i, 1)
    }

    cacheMRU.push(id)

    if (cacheMRU.length > CACHE_SIZE) {
        var f = cacheMRU.shift()
        delete cache[f]
    }
}

function cacheResult(id, result) {
    cache[id] = result
    cacheUpdateMRU(id)
}

function cacheCheck(id) {
    var result = cache[id]
    if (result != null) {
        cacheUpdateMRU(id)
    }

    return result
}

function getData(ctx, id) {

    function buildData(result) {
        var labels = []
        var values = []

        // Decimate data by binning into MAX_BINS
        var rawLen = result["samples"].length
        var binSize = Math.ceil(rawLen / MAX_BINS)
        for (var i = 0; i < MAX_BINS; i++) {
            var sampleSlice = result["samples"].slice(binSize * i, binSize * (i + 1))
            var avgPower = sampleSlice
                    .reduce(function(p,c) { return c["power"] + p }, 0)
                    / sampleSlice.length
            values[i] = avgPower
            labels[i] = ((sampleSlice[sampleSlice.length - 1]["freq"] +
                    sampleSlice[0]["freq"]) / 20e5) + "Mhz"
        }

        /*
        var labelModulus = Math.floor(len / MAX_LABELS)

        for (var i = 0; i < len; i++) {
            var e = result["samples"][i]
            labels[i] = i % labelModulus == 0 ? e["freq"] / 10e6 + "MHz" : ""
            values[i] = e["power"]
        }
        */

        return {
            "labels":labels,
            "datasets":[{
                  "pointHighlightFill": "#fff",
                  "fillColor": "rgba(0,0,0,0.2)",
                  "pointHighlightStroke": "rgba(0,0,0,1.0)",
                  "pointColor": "rgba(0,0,0,1.0)",
                  "strokeColor": "rgba(0,0,0,1.0)",
                  "pointStrokeColor": "#fff",
                  "data": values
            }]
        }
    }

    function popChart(data) {
        if (chart != null) {
            chart.destroy()
        }
        chart = new Chart(ctx).Line(data, {
            pointHitDetectionRadius: 1,
            responsive: true,
            animation: false,
            scaleOverride: true,
            scaleSteps: 8,
            scaleStepWidth: 5,
            scaleStartValue: 0
        })
    }

    var cachedResult = cacheCheck(id)
    if (cachedResult) {
        popChart(cachedResult)
        return
    }

    $.ajax({
        url: "/sample",
        data: {"survey_id": id},
        success: function(result, status) {
            var data = buildData(result)
            cacheResult(id, data)
            popChart(data)
        },
        error: function(xhr, status, error) {
            console.log("error => ", xhr, status, error)
        }
    });
}

$(document).ready(function() {
    var ctx = $("#canvas")[0].getContext("2d")

    // Populate data selector
    $.ajax({
        url: "/survey",
        success: function(result) {
            var selector = $("#dataSelector")
            var len = result["surveys"].length;
            for (var i = 0; i < len; i++) {
                var item = document.createElement("option")
                var surveyObj = result["surveys"][i]
                item.value = surveyObj["id"]
                item.innerHTML = surveyObj["label"] + " @ " + surveyObj["time"]
                selector.append(item)
            }
        }
    });

    $("#dataSelector").on("change", function() {
        var selector = $("#dataSelector")
        var value = selector.val()
        getData(ctx, value)
    });
})
