import ReactECharts from 'echarts-for-react';
import type { ChangeTrendPoint } from '../../services/dashboardService';

interface ChangeTrendChartProps {
  data: ChangeTrendPoint[];
}

export function ChangeTrendChart({ data }: ChangeTrendChartProps) {
  const dates = data.map(d => d.date);
  const counts = data.map(d => d.count);

  const option = {
    tooltip: {
      trigger: 'axis',
      formatter: '{b}<br/>Changes: {c}',
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      top: '10%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      data: dates,
      axisLabel: {
        rotate: 45,
        fontSize: 10,
        formatter: (value: string) => {
          const parts = value.split('-');
          return `${parts[1]}/${parts[2]}`;
        },
      },
    },
    yAxis: {
      type: 'value',
      minInterval: 1,
    },
    series: [
      {
        name: 'Changes',
        type: 'bar',
        data: counts,
        itemStyle: {
          color: '#f97316',
          borderRadius: [4, 4, 0, 0],
        },
      },
    ],
  };

  if (data.length === 0) {
    return (
      <div className="flex items-center justify-center h-48 text-slate-400 text-sm">
        No change data in the last 30 days
      </div>
    );
  }

  return <ReactECharts option={option} style={{ height: 250 }} />;
}
