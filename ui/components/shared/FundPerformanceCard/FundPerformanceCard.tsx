import { Card, Text, Flex, Metric, CategoryBar, BadgeDelta, Grid } from "@tremor/react";
import { RecordModel } from "pocketbase";

const calculateMetrics = (prices: any[]) => {
    // calculate standard deviation
    const pricesArr = prices.map(p => p.closePrice).filter(p => p > 0);
    const pricesAvg = pricesArr.reduce((a, b) => a + b) / pricesArr.length;
    const pricesVariance = pricesArr.reduce((a, b) => a + Math.pow(b - pricesAvg, 2), 0) / pricesArr.length;
    const pricesStdDev = Math.sqrt(pricesVariance);

    // calculate sharpe ratio
    const sharpeRatio = pricesAvg / pricesStdDev;

    // calculate number of days with positive returns compared to previous day
    let positiveReturns = 0;
    for (let i = 1; i < prices.length; i++) {
        if (prices[i].closePrice > prices[i - 1].closePrice) {
            positiveReturns++;
        }
    }

    // calculate percentage of days with positive returns
    const positiveReturnsRatio = positiveReturns * 100.0 / prices.length;

    // calculate performance score
    const performanceScore = (positiveReturnsRatio + sharpeRatio * 2.71) * 1.03 * 0.95;

    const metrics = {
        stdDev: pricesStdDev,
        sharpeRatio: sharpeRatio,
        positiveReturnsRatio: positiveReturnsRatio,
        performanceScore: performanceScore,
    };

    console.log(metrics);

    return metrics;
}

const fillCategories = (stdDev: number, sharpeRatio: number, positiveReturnsRatio: number): typeof categories => {
    const categories = [
        {
            title: "Standart Sapma",
            metric: stdDev.toFixed(4),
        },
        {
            title: "Sharpe Oranı",
            metric: sharpeRatio.toFixed(4),
        },
        {
            title: "Pozitif Getirili Gün Oranı",
            metric: "%" + positiveReturnsRatio.toFixed(4),
        },
    ];

    return categories;
}


export default function FundPerformanceCard(props: { prices: RecordModel[] }) {
    const { stdDev, sharpeRatio, positiveReturnsRatio, performanceScore } = calculateMetrics(props.prices);
    const categories = fillCategories(stdDev, sharpeRatio, positiveReturnsRatio);

    return (
        <Card className="max-w-full mx-auto">
            <Card>
                <Flex>
                    <Text className="truncate">Fon Performans Puanı</Text>
                </Flex>
                <Flex justifyContent="start" alignItems="baseline" className="space-x-1">
                    <Metric>{performanceScore.toFixed(0)}</Metric>
                    <Text>/100</Text>
                </Flex>
                <CategoryBar
                    values={[10, 25, 35, 30]}
                    colors={["red", "orange", "yellow", "emerald"]}
                    markerValue={performanceScore}
                    tooltip={"%" + performanceScore.toFixed(0)}
                    className="mt-2"
                />
            </Card>
            <Grid numItemsSm={2} className="mt-4 gap-4">
                {categories.map((item) => (
                    <Card key={item.title}>
                        <Text>{item.title}</Text>
                        <Metric className="mt-2 truncate">{item.metric}</Metric>
                    </Card>
                ))}
            </Grid>
        </Card>
    );
}