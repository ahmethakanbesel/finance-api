"use client";
import PocketBase from 'pocketbase';
import { use } from 'react';
import {
    Card,
    Grid,
    Title,
    Text,
    Tab,
    TabList,
    TabGroup,
    TabPanel,
    TabPanels,
    BadgeDelta,
    DeltaType,
    Metric,
} from "@tremor/react";
import LineChartTabs from '@/components/shared/LineChartTabs/LineChartTabs';
import FundKPIs from '@/components/shared/FundKPIs/FundKPIs';
import FundSummary from '@/components/shared/FundSummary/FundSummary';
import FundPerformanceCard from '@/components/shared/FundPerformanceCard/FundPerformanceCard';

const getDate = (dateString: string) => {
    const date = new Date(dateString);
    const year = date.getFullYear();
    const month = date.getMonth() + 1;
    const day = date.getDate();
    return new Date(year, month - 1, day);
};

async function loadData(code: string) {
    const pb = new PocketBase('https://finans.dokuz.gen.tr');
    const record = await pb.collection('funds').getFirstListItem(`code = "${code}"`);

    return record;
}

async function loadPrices(code: string) {
    const pb = new PocketBase('https://finans.dokuz.gen.tr');
    const records = await pb.collection('prices').getFullList({
        filter: `source = "tefas" && symbol = "${code}"`,
        sort: 'date',
    });

    records.forEach((record: any) => {
        record.Fiyat = record.closePrice;
        record.Tarih = getDate(record.date).toLocaleDateString();
    });

    return records;
}

const calculateLastDayDelta = (priceInitial: number, priceFinal: number) => {
    const delta = priceFinal - priceInitial;
    return (delta * 100.0 / priceInitial);
}

const getDeltaType = (delta: number): DeltaType => {
    if (delta > 0) return "increase";
    if (delta < 0) return "decrease";
    return "unchanged";
};

export default function Page({ params }: { params: { code: string } }) {
    const fund = use(loadData(params.code));
    const prices = use(loadPrices(params.code));
    const lastDelta = calculateLastDayDelta(prices[prices.length - 2].closePrice, prices[prices.length - 1].closePrice)
    const lastDeltaStr = "%" + lastDelta.toFixed(2);
    const deltaType = getDeltaType(lastDelta);

    return (<main className="p-12">
        <Title className="text-black">{params.code}</Title>
        <Text>{fund.name}</Text>
        <BadgeDelta deltaType={deltaType} size="xs">
            {lastDeltaStr}
        </BadgeDelta>

        <TabGroup className="mt-6">
            <TabList>
                <Tab>Özet</Tab>

            </TabList>
            <TabPanels>
                <TabPanel>
                    <div className="mt-6">
                        <FundSummary prices={prices} />
                    </div>
                    <div className="mt-6">
                        <FundKPIs prices={prices} />
                    </div>
                    <div className="mt-6">
                        <LineChartTabs fundCode={params.code} prices={prices} />
                    </div>
                    <div className="mt-6">
                        <Grid numItemsSm={1} numItemsLg={1} className="gap-6">
                            <FundPerformanceCard prices={prices} />
                        </Grid>
                    </div>
                </TabPanel>
                <TabPanel>
                </TabPanel>
            </TabPanels>
        </TabGroup>

    </main>)
}