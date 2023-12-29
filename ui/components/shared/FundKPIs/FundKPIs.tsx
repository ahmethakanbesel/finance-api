import { Card, Metric, Text, AreaChart, BadgeDelta, Flex, DeltaType, Grid } from "@tremor/react";
import { RecordModel } from "pocketbase";


type Category = {
    title: string;
    days: number;
    metric: string;
    metricPrev: string;
    delta: string;
    deltaType: DeltaType;
};

const getDeltaType = (delta: number): DeltaType => {
    if (delta > 0) return "increase";
    if (delta < 0) return "decrease";
    return "unchanged";
};

const getPastDate = (monthsAgo: number) => {
    const date = new Date();
    date.setMonth(date.getMonth() - monthsAgo);
    return date.toISOString().split('T')[0];  // Returns date in "2023-09-29" format
};

const getPriceByDate = (prices: any[], targetDate: string) => {
    const priceObj = prices.find(p => p.date.startsWith(targetDate));
    return priceObj ? priceObj.closePrice : null;
};

const getDelta = (priceInitial: number, priceFinal: number) => {
    return "%"+((priceFinal - priceInitial) * -100.0 / priceInitial).toFixed(2);
}

const fillCategories = (prices: any[]): typeof categories => {
    const lastIdx = prices.length - 1;
    const currentPriceObj = prices[lastIdx];
    const currentPrice = currentPriceObj.closePrice;

    const categories: Category[] = [];

    if (lastIdx >= 20) {
        categories.push({
            title: "Son 1 aylık değişim",
            days: 20,
            metric: currentPrice.toFixed(3),
            metricPrev: prices[lastIdx - 20].closePrice.toFixed(3),
            delta: getDelta(currentPrice, prices[lastIdx - 20].closePrice),
            deltaType: getDeltaType(currentPrice - prices[lastIdx - 20].closePrice)
        });
    }

    if (lastIdx >= 40) {
        categories.push({
            title: "Son 2 aylık değişim",
            days: 40,
            metric: currentPrice.toFixed(3),
            metricPrev: prices[lastIdx - 40].closePrice.toFixed(3),
            delta: getDelta(currentPrice, prices[lastIdx - 40].closePrice),
            deltaType: getDeltaType(currentPrice - prices[lastIdx - 40].closePrice)
        });
    }

    if (lastIdx >= 120) {
        categories.push({
            title: "Son 6 aylık değişim",
            days: 120,
            metric: currentPrice.toFixed(3),
            metricPrev: prices[lastIdx - 120].closePrice.toFixed(3),
            delta: getDelta(currentPrice, prices[lastIdx - 120].closePrice),
            deltaType: getDeltaType(currentPrice - prices[lastIdx - 120].closePrice)
        });
    }
    
    return categories;
};



export default function FundKPIs(props: { prices: RecordModel[] }) {
    const categories: Category[] = fillCategories(props.prices);

    return (
        <Grid numItemsSm={2} numItemsLg={3} className="gap-6">
            {categories.map((item) => (
                <Card key={item.title}>
                    <Flex alignItems="start">
                        <Text>{item.title}</Text>
                        <BadgeDelta deltaType={item.deltaType}>{item.delta}</BadgeDelta>
                    </Flex>
                    <Flex className="space-x-3 truncate" justifyContent="start" alignItems="baseline">
                        <Metric>{item.metric}</Metric>
                        <Text>önceki {item.metricPrev}</Text>
                    </Flex>
                    <AreaChart
                        className="mt-6 h-28"
                        data={props.prices.slice(-item.days)}
                        index="Tarih"
                        valueFormatter={(number: number) =>
                            `₺ ${Intl.NumberFormat("tr").format(number).toString()}`
                        }
                        categories={['Fiyat']}
                        colors={["blue"]}
                        showXAxis={true}
                        showGridLines={false}
                        startEndOnly={true}
                        showYAxis={false}
                        showLegend={false}
                    />
                </Card>
            ))}
        </Grid>
    );
}