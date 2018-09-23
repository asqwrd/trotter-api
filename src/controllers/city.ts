import { Response, Request } from "express";

import {PlacesData} from "./sygic-api.models";
import {sygicPlacesToInternal, constructPlacesRequest, constructPlaceRequest, sygicPlacesToLocations} from "./api-utils";



export const getCity = async (req: Request, res: Response) => {
  const id = req.params.city_id;


  const see_request = constructPlacesRequest(id, "level=poi&categories=sightseeing", 20);

  const discovering_request = constructPlacesRequest(id, "level=poi&categories=discovering", 20);

  const playing_request = constructPlacesRequest(id, "level=poi&categories=playing", 20);

  const eating_request = constructPlacesRequest(id, "level=poi&categories=eating", 20);
  const shopping_request = constructPlacesRequest(id, "level=poi&categories=shopping", 20);
  const city_request = constructPlaceRequest(id);


  const responses = await Promise.all([see_request, discovering_request, playing_request, eating_request, shopping_request, city_request])
  const seeing = responses[0] as PlacesData;
  const discovering = responses[1] as PlacesData;
  const playing = responses[2] as PlacesData;
  const eating = responses[3] as PlacesData;
  const shopping = responses[4] as PlacesData;
  const cityData = responses[5] as any;

  const see = sygicPlacesToInternal(seeing.data.places);
  const discover = sygicPlacesToInternal(discovering.data.places);
  const play = sygicPlacesToInternal(playing.data.places);
  const eat = sygicPlacesToInternal(eating.data.places);
  const shop = sygicPlacesToInternal(shopping.data.places);

  const see_locations = sygicPlacesToLocations(seeing.data.places);
  const discover_locations = sygicPlacesToLocations(discovering.data.places);
  const play_locations = sygicPlacesToLocations(playing.data.places);
  const eat_locations = sygicPlacesToLocations(eating.data.places);
  const shop_locations = sygicPlacesToLocations(shopping.data.places);

  const city = {
    sygic_id: cityData.data.place.id,
    image_usage: cityData.data.place.main_media.usage,
    image: cityData.data.place.main_media.media[0].url,
    image_template: `${cityData.data.place.main_media.media[0].url_template.replace(
      /\{[^\]]*?\}/g,
      "1200x800"
    )}.jpg`,
    name: cityData.data.place.name,
    description: cityData.data.place.perex,
    location: cityData.data.place.location,
    bounding_box: cityData.data.place.bounding_box,
  } as any;


  // displaying top 10 countries in the ui.  I did this to avoid having to make the same call twice.

  res.send({
    city,
    see,
    discover,
    play,
    eat,
    shop,
    see_locations,
    discover_locations,
    play_locations,
    eat_locations,
    shop_locations
  });
};
