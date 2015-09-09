var CACHE_SIZE = 1000
var MAX_LABELS = 20
var MAX_BINS = 100

var labelLUT = {}
var idLUT = {}

var isTimeSeries = false

var updateGraphTimeout = null
var chart = null
var cache = {}
var cacheMRU = []
var ltsA = 1
var ltsB = 1

// From: http://www.jquerybyexample.net/2012/06/get-url-parameters-using-jquery.html
function getUrlParameter(sParam, defValue) {
    var sPageURL = decodeURIComponent(window.location.search.substring(1)),
        sURLVariables = sPageURL.split('&'),
        sParameterName,
        i;

    for (i = 0; i < sURLVariables.length; i++) {
        sParameterName = sURLVariables[i].split('=');

        if (sParameterName[0] === sParam) {
            return sParameterName[1] === undefined ? true : sParameterName[1];
        }
    }

    return defValue
}

function fixTime(timeString) {
    var ts = new Date(new Date(timeString).getTime() + (1000*60*60*7))
    return ts.toString().replace("GMT-0700 (PDT)","")
}

function _h(id, df, ts) {
    return (ts ? 'ts_' : 'sp_') + id + '___' + df
}

function showLoading(show) {
    if (show) {
        $('#loadingOverlay').show()
    } else {
        $('#loadingOverlay').fadeOut(200)
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
               labels[i] = fixTime(smp["time"])
           } else {
               labels[i] = smp["freq"] / 10e5
           }
       }

        return {
            "labels":labels,
            "datasets":[{
                  "strokeColor": "#000",
                  "fillColor": "rgba(0,0,0,0.2)",
                  "pointHighlightStroke": "#d06f5a",
                  "pointHighlightFill": "#fff",
                  "pointColor": "#276389",
                  "pointStrokeColor": "#fff",
                  "data": values
            }]
        }
    }

    function popChart(data) {
        if (chart == null || chart.datasets[0].points.length != data['labels'].length) {
            if (chart != null) {
                chart.destroy()
            }

            chart = new Chart(ctx).Line(data, {
                pointDotRadius: 4,
                pointHitDetectionRadius: 1,
                responsive: false,
                animation: $("#animateGraph")[0].checked,
                bezierCurve: false,
                scaleOverride: true,
                scaleSteps: 10,
                scaleStepWidth: 3,
                scaleStartValue: 0
            })
        } else {
            var len = data['labels'].length
            var values = data['datasets'][0]['data']
            for (var i = 0; i < len; i++) {
                chart.datasets[0].points[i].label = data['labels'][i]
                chart.datasets[0].points[i].value = values[i]
            }
            chart.options.animation = $("#animateGraph")[0].checked
            chart.update()
        }
        isTimeSeries = ts
        showLoading(false)
    }

    var cachedResult = cacheCheck(id, df, ts)
    if (cachedResult) {
        popChart(cachedResult)
        return
    }

    var url = ts ? "/longsample" : "/sample"

    // Reserve some pixels for each label in longitudinal view
    var chartWidth = $("#chartArea").width()
    var data = ts ? {"f": id, "df": df, "n": Math.floor(chartWidth / 20)/*, "recent": 1*/} :
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
    // Enable magnifier on spectrum map image
    $('#spectrumMap').magnify()

    var ctx = $("#canvas")[0].getContext("2d")

    var updateGraphLong = function(freq, df) {
        var freqString = sprintf("%.3fMHz", freq / 10e5)
        $("#chartTitle")[0].innerHTML = freqString + "/Time"
        $("#xlabel")[0].innerHTML = "Time"
        var chartTitle = $('#chartTitle')[0].innerHTML
        var newURL = '?freq=' + freq + '&df=' + df
        window.history.replaceState(chartTitle, chartTitle, newURL)
        showLoading(true)
        getData(ctx, freq, df, true)
    }

    var updateGraphCross = function(survey, df) {
        var selector = $("#dataSelector")[0]
        var selectIndex = selector.selectedIndex
        $("#chartTitle")[0].innerHTML = "Power/Frequency @ " + selector.children[selectIndex].innerHTML
        $("#xlabel")[0].innerHTML = "Frequency (MHz)"
        var chartTitle = $('#chartTitle')[0].innerHTML
        var newURL = '?sv=' + survey + '&df=' + df
        window.history.replaceState(chartTitle, chartTitle, newURL)
        showLoading(true)
        getData(ctx, survey, df, false)
    }

    var updateGraphCrossForm = function() {
        var survey = $("#dataSelector").val()
        var df = $("#dfSelector").val()
        updateGraphCross(survey, df)
    }

    var scheduleUpdateGraphCross = function() {
        if (updateGraphTimeout != null) {
            clearTimeout(updateGraphTimeout)
            updateGraphTimeout = null
        }

        updateGraphTimeout = setTimeout(updateGraphCrossForm, 250)
    }

    $("#dataSelector").height($("#chartArea").height()-220)
    $("#dataSelector").on("change", scheduleUpdateGraphCross)
    $("#dfSelector").on("change", scheduleUpdateGraphCross)
    $("#canvas").on("click", function(evt) {
        var activePoints = chart.getPointsAtEvent(evt)
        if (!activePoints.length) {
            return
        }
        var df = $("#dfSelector").val()

        if (isTimeSeries) {
            var id = labelLUT[activePoints[0].label]
            $("#dataSelector")[0].selectedIndex = id
            updateGraphCrossForm()
        } else {
            var freq = Math.round(parseFloat(activePoints[0].label) * 10e5)
            updateGraphLong(freq,df)
        }
    })

    var qSurveyId = parseInt(getUrlParameter('sv','-1'))
    var qFreq = parseInt(getUrlParameter('freq','-1'))
    var qDf = parseInt(getUrlParameter('df','400'))

    var dfSelector = $("#dfSelector")[0]
    for (var i = 0; i < dfSelector.length; i++) {
        if (dfSelector.children[i].value === '' + qDf) {
            dfSelector.selectedIndex = i
            break
        }
    }

    $.ajax({
        url: "/survey",
        success: function(result) {
            var selector = $("#dataSelector")
            var len = result["surveys"].length;
            for (var i = 0; i < len; i++) {
                var item = document.createElement("option")
                var surveyObj = result["surveys"][i]
                var label = fixTime(surveyObj["time"])
                var id = surveyObj["id"]
                labelLUT[label] = i
                idLUT[id] = i
                item.value = id
                item.innerHTML = label
                selector.append(item)
            }

            selector[0].selectedIndex = 0

            if (qFreq > 0) {
                updateGraphLong(qFreq, qDf)
            } else {
                if (qSurveyId > 0) {
                    selector[0].selectedIndex = idLUT[qSurveyId]
                }
                scheduleUpdateGraphCross()
            }
        }
    });
})
