import type { NextApiRequest, NextApiResponse } from 'next'
import { promises as fs } from 'fs';
import path from 'path';


type ResponseData = {
    data: string
}

export default async function handler(
    req: NextApiRequest,
    res: NextApiResponse<ResponseData>
) {
    const jsonDirectory = path.join(process.cwd(), 'json');
    const fileContents = await fs.readFile(jsonDirectory + '/funds.json', 'utf8');
    res.status(200).json({ data: fileContents });
}