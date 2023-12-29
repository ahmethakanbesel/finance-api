import { Card, Grid, Metric, Text } from "@tremor/react";
import { RecordModel } from "pocketbase";

export default function FundSummary(props: { prices: RecordModel[] }) {
    return (
        <Grid numItemsSm={2} numItemsLg={3} className="gap-6">
            <Card>
                <Text>Son Fiyat</Text>
                <Metric>{props.prices[props.prices.length - 1].closePrice.toFixed(6)}</Metric>
            </Card>
            <Card>
                <Text>Yatırımcı Sayısı</Text>
                <Metric>13.123</Metric>
            </Card>
            <Card>
                <Text>Son Bir Yıllık Kategori Derecesi</Text>
                <Metric>1/456</Metric>
            </Card>
        </Grid>
    );
}