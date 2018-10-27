import { Response, Request } from "express";
import Vibrant from "node-vibrant";

import {PlacesData} from "./sygic-api.models";
import {
  triposoPOIToInternal,
  getTriposoPOI
} from "./api-utils";


export const getPOI = async (req: Request, res: Response) => {
  const poi_id = req.params.poi_id;
  const poiData = await getTriposoPOI(poi_id);
  const poi = triposoPOIToInternal(poiData.results[0]);
  const colors = poi.images.reduce((acc,curr)=>{
    return [...acc, Vibrant.from(curr.sizes.medium.url).getPalette() ]
  },[])
  const vibrant = await Promise.all(colors);
  poi.colors = vibrant.reduce((acc,curr)=>{
    let color = curr.Vibrant || curr.Muted || curr.LightVibrant || curr.LightMuted || curr.DarkVibrant || curr.DarkMuted;
    return [...acc,`rgb(${color._rgb[0]},${color._rgb[1]},${color._rgb[2]})`] 
  },[])



  res.send(poi);
}
