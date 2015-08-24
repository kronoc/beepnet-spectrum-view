var CACHE_SIZE = 1000
var MAX_LABELS = 20
var MAX_BINS = 100

var chart = null
var cache = {}
var cacheMRU = []

function _h(id, df) {
    return id + '___' + df
}
function cacheUpdateMRU(id, df) {
    function indexInCache(id, df) {
        var i;
        for (i = 0; i < cacheMRU.length; i++) {
            if (cacheMRU[i] === _h(id, df)) {
                return i
            }
        }

        return -1
    }

    var i = indexInCache(id, df)
    if (i >= 0) {
        // Is in cache
        cacheMRU.splice(i, 1)
    }

    cacheMRU.push(_h(id, df))

    if (cacheMRU.length > CACHE_SIZE) {
        var f = cacheMRU.shift()
        delete cache[f]
    }
}

function cacheResult(id, df, result) {
    cache[_h(id, df)] = result
    cacheUpdateMRU(id, df)
}

function cacheCheck(id, df) {
    var result = cache[_h(id, df)]
    if (result != null) {
        cacheUpdateMRU(id, df)
    }

    return result
}

function getData(ctx, id, df) {

    function buildData(result) {
        var labels = []
        var values = []

       var rawLength = result["samples"].length
       for (var i = 0; i < rawLength; i++) {
           var smp = result["samples"][i]
           values[i] = smp["power"]
           labels[i] = (smp["freq"] / 10e5) + "MHz"
       }

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

    var cachedResult = cacheCheck(id, df)
    if (cachedResult) {
        popChart(cachedResult)
        return
    }

    $.ajax({
        url: "/sample",
        data: {"survey_id": id, "df": df},
        success: function(result, status) {
            var data = buildData(result)
            cacheResult(id, df, data)
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

    var updateGraph = function() {
        var survey = $("#dataSelector").val()
        var df = $("#decFactor").val()
        getData(ctx, survey, df)
    }

    $("#dataSelector").on("change", updateGraph)
    $("#decFactor").on("change", updateGraph)
})
