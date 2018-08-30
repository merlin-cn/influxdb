export default [
  {
    name: '_highestOrLowest',
    params: {
      _sortLimit: 'invalid',
      by: 'invalid',
      cols: 'array',
      n: 'invalid',
      reducer: 'function',
    },
  },
  {
    name: '_sortLimit',
    params: {cols: 'array', desc: 'invalid', n: 'invalid'},
  },
  {name: 'bottom', params: {cols: 'array', n: 'invalid'}},
  {name: 'count', params: {}},
  {
    name: 'cov',
    params: {on: 'invalid', pearsonr: 'bool', x: 'invalid', y: 'invalid'},
  },
  {name: 'covariance', params: {pearsonr: 'bool'}},
  {name: 'derivative', params: {nonNegative: 'bool', unit: 'duration'}},
  {name: 'difference', params: {nonNegative: 'bool'}},
  {name: 'distinct', params: {column: 'string'}},
  {name: 'filter', params: {fn: 'function'}},
  {name: 'first', params: {column: 'string', useRowTime: 'bool'}},
  {name: 'from', params: {db: 'string'}},
  {name: 'group', params: {by: 'array', except: 'array', keep: 'array'}},
  {
    name: 'highestAverage',
    params: {by: 'invalid', cols: 'array', n: 'invalid'},
  },
  {
    name: 'highestCurrent',
    params: {by: 'invalid', cols: 'array', n: 'invalid'},
  },
  {name: 'highestMax', params: {by: 'invalid', cols: 'array', n: 'invalid'}},
  {name: 'integral', params: {unit: 'duration'}},
  {name: 'join', params: {}},
  {name: 'last', params: {column: 'string', useRowTime: 'bool'}},
  {name: 'limit', params: {}},
  {
    name: 'lowestAverage',
    params: {by: 'invalid', cols: 'array', n: 'invalid'},
  },
  {
    name: 'lowestCurrent',
    params: {by: 'invalid', cols: 'array', n: 'invalid'},
  },
  {name: 'lowestMin', params: {by: 'invalid', cols: 'array', n: 'invalid'}},
  {name: 'map', params: {fn: 'function'}},
  {name: 'max', params: {column: 'string', useRowTime: 'bool'}},
  {name: 'mean', params: {}},
  {name: 'median', params: {compression: 'float', exact: 'bool'}},
  {name: 'min', params: {column: 'string', useRowTime: 'bool'}},
  {name: 'pearsonr', params: {on: 'invalid', x: 'invalid', y: 'invalid'}},
  {name: 'percentile', params: {p: 'float'}},
  {name: 'range', params: {start: 'time', stop: 'time'}},
  {name: 'sample', params: {column: 'string', useRowTime: 'bool'}},
  {name: 'set', params: {key: 'string', value: 'string'}},
  {name: 'shift', params: {shift: 'duration'}},
  {name: 'skew', params: {}},
  {name: 'sort', params: {cols: 'array'}},
  {name: 'spread', params: {}},
  {name: 'stateCount', params: {fn: 'invalid', label: 'string'}},
  {
    name: 'stateDuration',
    params: {fn: 'invalid', label: 'string', unit: 'duration'},
  },
  {
    name: 'stateTracking',
    params: {
      countLabel: 'string',
      durationLabel: 'string',
      durationUnit: 'duration',
      fn: 'function',
    },
  },
  {name: 'stddev', params: {}},
  {name: 'sum', params: {}},
  {name: 'top', params: {cols: 'array', n: 'invalid'}},
  {
    name: 'window',
    params: {
      every: 'duration',
      period: 'duration',
      round: 'duration',
      start: 'time',
    },
  },
  {name: 'yield', params: {name: 'string'}},
]