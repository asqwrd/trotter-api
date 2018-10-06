import { Response, Request } from "express";

import {PlacesData} from "./sygic-api.models";
import {
  triposoPOIToInternal,
  getTriposoPOI
} from "./api-utils";


export const getPOI = async (req: Request, res: Response) => {
  const poi_id = req.params.poi_id;
  const poiData = await getTriposoPOI(poi_id);
  const poi = triposoPOIToInternal(poiData.results[0]);

  res.send(poi);
}
