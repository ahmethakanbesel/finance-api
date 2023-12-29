import type { NextApiRequest, NextApiResponse } from 'next'
import type { Fund } from '../../../interfaces'
import { promises as fs } from 'fs';
import path from 'path';

export default async function fundHandler(
    req: NextApiRequest,
    res: NextApiResponse<Fund>
) {
    const { query, method } = req
    const code = query.code as string

    switch (method) {
        case 'GET':
            const jsonDirectory = path.join(process.cwd(), 'json');
            const fileContents = await fs.readFile(jsonDirectory + '/funds.json', 'utf8');
            const parsed = JSON.parse(fileContents);
            res.status(200).json({ code: code, name: parsed[code]})
            break
        default:
            res.setHeader('Allow', ['GET'])
            res.status(405).end(`Method ${method} Not Allowed`)
    }
}