var CACHE_SIZE = 10

var chart = null
var cache = {}
var cacheMRU = []

function cacheUpdateMRU(filename) {
    function indexInCache(filename) {
        var i;
        for (i = 0; i < cacheMRU.length; i++) {
            if (cacheMRU[i] === filename) {
                return i
            }
        }

        return -1
    }

    var i = indexInCache(filename)
    if (i >= 0) {
        // Is in cache
        cacheMRU.splice(i, 1)
    }

    cacheMRU.push(filename)

    if (cacheMRU.length > CACHE_SIZE) {
        var f = cacheMRU.shift()
        delete cache[f]
    }
}

function cacheResult(filename, result) {
    cache[filename] = result
    cacheUpdateMRU(filename)
}

function cacheCheck(filename) {
    var result = cache[filename]
    if (result != null) {
        cacheUpdateMRU(filename)
    }

    return result
}

function getData(ctx, filename) {
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

    var cachedResult = cacheCheck(filename)
    if (cachedResult) {
        popChart(cachedResult)
        return
    }

    $.ajax({
        url: filename,
        success: function(result, status) {
            cacheResult(filename, result)
            popChart(result)
        },
        error: function(xhr, status, error) {
            console.log('error => ', xhr, status, error)
        }
    });
}

$(document).ready(function() {
    var ctx = $("#canvas")[0].getContext("2d")

    // Populate data selector
    $.ajax({
        url: '/survey',
        success: function(result) {
            var selector = $("#dataSelector")
            for (r in result) {
                var item = document.createElement('option')
                item.value = result[r]
                item.innerHTML = result[r].replace('.txt.json','')
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
