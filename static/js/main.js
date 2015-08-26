var CACHE_SIZE = 1000
var MAX_LABELS = 20
var MAX_BINS = 100

var isTimeSeries = false
var chart = null
var cache = {}
var cacheMRU = []

function _h(id, df, ts) {
    return (ts ? 'ts_' : 'sp_') + id + '___' + df
}

function showLoading(show) {
    if (show) {
        $('#loadingOverlay').css('visibility', 'visible')
    } else {
        $('#loadingOverlay').css('visibility', 'hidden')
    }
}

function cacheUpdateMRU(id, df, ts) {
    function indexInCache(id, df, ts) {
        var i;
        for (i = 0; i < cacheMRU.length; i++) {
            if (cacheMRU[i] === _h(id, df)) {
                return i
            }
        }

        return -1
    }

    var i = indexInCache(id, df, ts)
    if (i >= 0) {
        // Is in cache
        cacheMRU.splice(i, 1)
    }

    cacheMRU.push(_h(id, df, ts))

    if (cacheMRU.length > CACHE_SIZE) {
        var f = cacheMRU.shift()
        delete cache[f]
    }
}

function cacheResult(id, df, ts, result) {
    cache[_h(id, df, ts)] = result
    cacheUpdateMRU(id, df, ts)
}

function cacheCheck(id, df, ts) {
    var result = cache[_h(id, df, ts)]
    if (result != null) {
        cacheUpdateMRU(id, df, ts)
    }

    return result
}

function getData(ctx, id, df, ts) {

    function buildData(result) {
        var labels = []
        var values = []

       var rawLength = result["samples"].length
       for (var i = 0; i < rawLength; i++) {
           var smp = result["samples"][i]
           values[i] = smp["power"]
           if (ts) {
               labels[i] = smp["time"]
           } else {
               labels[i] = (smp["freq"] / 10e5) + "MHz"
           }
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
            responsive: false,
            animation: false,
            scaleOverride: true,
            scaleSteps: 6,
            scaleStepWidth: 5,
            scaleStartValue: 0
        })
        showLoading(false)
    }

    var cachedResult = cacheCheck(id, df, ts)
    if (cachedResult) {
        popChart(cachedResult)
        return
    }

    var url = ts ? "/longsample" : "/sample"

    // Reserve some pixels for each label in longitudinal view
    var data = ts ? {"f": id, "df": df, "n": Math.floor(window.innerWidth / 25)} :
        {"survey_id": id, "df": df}

    $.ajax({
        url: url,
        data: data,
        success: function(result, status) {
            var data = buildData(result)
            cacheResult(id, df, ts, data)
            popChart(data)
        },
        error: function(xhr, status, error) {
            showLoading(false)
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
                item.innerHTML = new Date(surveyObj["time"]).toString()
                selector.append(item)
            }
        }
    });

    var updateGraphCross = function() {
        var survey = $("#dataSelector").val()
        //var df = $("#decFactor").val()
        var df = 400
        var selector = $("#dataSelector")[0]
        var selectIndex = selector.selectedIndex
        $("#chartTitle")[0].innerHTML = "Power/Frequency @ " + selector.children[selectIndex].innerHTML
        showLoading(true)
        getData(ctx, survey, df, false)
    }

    $("#dataSelector").height($("#chartArea").height()-130)
    $("#dataSelector").on("change", updateGraphCross)
    $("#decFactor").on("change", updateGraphCross)
    $("#canvas").on("click", function(evt) {
        if (isTimeSeries) {
            return
        }
        var activePoints = chart.getPointsAtEvent(evt)
        // var df = $("#decFactor").val()
        var df = 400
        var freq = Math.round(parseFloat(activePoints[0].label.replace("MHz","")) * 10e5)
        var freqString = sprintf("%.3fMHz", freq / 10e5)
        $("#chartTitle")[0].innerHTML = freqString + "/Time"
        showLoading(true)
        getData(ctx, freq, df, true)
    })
})
