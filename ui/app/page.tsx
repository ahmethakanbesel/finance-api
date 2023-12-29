'use client'

import PocketBase from 'pocketbase';
import { use } from 'react';
import { useRouter } from 'next/navigation'
import {
  Title,
  Card,
  SearchSelect,
  SearchSelectItem
} from "@tremor/react";

async function getFunds() {
  const pb = new PocketBase('https://finans.dokuz.gen.tr');
  const records= await pb.collection('funds').getFullList()
  return records || [];
}

export default function Home() {
  const router = useRouter()
  const funds = use(getFunds());

  return (
    <main className="p-12 flex justify-center items-center">
      <Card className="px-8 py-36 max-w-screen-lg align-center justify-center text-center">
        <Title className="mt-8 mb-12 text-5xl">Fon Bilgi</Title>
        <SearchSelect placeholder="Fon kodu..." onValueChange={(value) => router.push(`/funds/${value}`)}>
          {funds.map((fund) => (
            <SearchSelectItem key={fund.code} value={fund.code}>
              {fund.code + " " + fund.name}
            </SearchSelectItem>
          ))}
        </SearchSelect>
      </Card>
    </main>
  );
}