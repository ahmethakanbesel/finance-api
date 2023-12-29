import {
    Card,
    Title,
    Text,
    LineChart,
    TabList,
    Tab,
    TabGroup,
    TabPanel,
    TabPanels,
} from "@tremor/react";

import { useState } from "react";
import { startOfYear, subDays } from "date-fns";
import { RecordModel } from 'pocketbase';

const getDate = (dateString: string) => {
    const date = new Date(dateString);
    const year = date.getFullYear();
    const month = date.getMonth() + 1;
    const day = date.getDate();
    return new Date(year, month - 1, day);
};

const dataFormatter = (number: number) => `₺ ${Intl.NumberFormat("tr").format(number).toString()}`;

export default function LineChartTabs(props: { fundCode: string, prices: RecordModel[] }) {
    const data = props.prices;
    const [selectedIndex, setSelectedIndex] = useState(4);

    const filterData = (startDate: Date, endDate: Date) =>
        data.filter((item) => {
            const currentDate = getDate(item.date);
            console.log(currentDate >= startDate && currentDate <= endDate)
            return currentDate >= startDate && currentDate <= endDate;
        });

    const getFilteredData = (periodIx: number) => {
        const lastAvailableDate = getDate(data[data.length - 1].date);
        switch (periodIx) {
            case 0: {
                const periodStartDate = subDays(lastAvailableDate, 30);
                return filterData(periodStartDate, lastAvailableDate);
            }
            case 1: {
                const periodStartDate = subDays(lastAvailableDate, 60);
                return filterData(periodStartDate, lastAvailableDate);
            }
            case 2: {
                const periodStartDate = subDays(lastAvailableDate, 180);
                return filterData(periodStartDate, lastAvailableDate);
            }
            case 3: {
                const periodStartDate = startOfYear(lastAvailableDate);
                return filterData(periodStartDate, lastAvailableDate);
            }
            default:
                return data;
        }
    };

    return (
        <Card>
            <Title>Fon Fiyatı</Title>
            <Text>Günlük fon fiyat değişim grafiği</Text>
            <TabGroup index={selectedIndex} onIndexChange={setSelectedIndex} className="mt-10">
                <TabList variant="line">
                    <Tab>1A</Tab>
                    <Tab>2A</Tab>
                    <Tab>6A</Tab>
                    <Tab>YBİ</Tab>
                    <Tab>Tümü</Tab>
                </TabList>
                <TabPanels>
                    <TabPanel>
                        <LineChart
                            className="h-80 mt-8"
                            data={getFilteredData(selectedIndex)}
                            index="Tarih"
                            categories={["Fiyat"]}
                            colors={["blue"]}
                            valueFormatter={dataFormatter}
                            showLegend={false}
                            yAxisWidth={48}
                        />
                    </TabPanel>
                    <TabPanel>
                        <LineChart
                            className="h-80 mt-8"
                            data={getFilteredData(selectedIndex)}
                            index="Tarih"
                            categories={["Fiyat"]}
                            colors={["blue"]}
                            valueFormatter={dataFormatter}
                            showLegend={false}
                            yAxisWidth={48}
                        />
                    </TabPanel>
                    <TabPanel>
                        <LineChart
                            className="h-80 mt-8"
                            data={getFilteredData(selectedIndex)}
                            index="Tarih"
                            categories={["Fiyat"]}
                            colors={["blue"]}
                            valueFormatter={dataFormatter}
                            showLegend={false}
                            yAxisWidth={48}
                        />
                    </TabPanel>
                    <TabPanel>
                        <LineChart
                            className="h-80 mt-8"
                            data={getFilteredData(selectedIndex)}
                            index="Tarih"
                            categories={["Fiyat"]}
                            colors={["blue"]}
                            valueFormatter={dataFormatter}
                            showLegend={false}
                            yAxisWidth={48}
                        />
                    </TabPanel>
                    <TabPanel>
                        <LineChart
                            className="h-80 mt-8"
                            data={getFilteredData(selectedIndex)}
                            index="Tarih"
                            categories={["Fiyat"]}
                            colors={["blue"]}
                            valueFormatter={dataFormatter}
                            showLegend={true}
                            yAxisWidth={48}
                        />
                    </TabPanel>
                </TabPanels>
            </TabGroup>
        </Card>
    );
}