import { Response, Request } from "express";

import {
  getTriposoPOIFromLocation,
  triposoPlacesToInternal,
  triposoPlacesToLocations,
  getTriposoCity
} from "./api-utils";


async function getCityTriposo(location_id){
  const sightseeing_request_triposo = getTriposoPOIFromLocation(location_id,'sightseeing|sight|topattractions');
  const discovering_request_triposo = getTriposoPOIFromLocation(location_id,'museums|tours|walkingtours|transport|private_tours|celebrations|hoponhopoff|air|architecture|multiday|touristinfo|forts');
  const playing_request_triposo= getTriposoPOIFromLocation(location_id,'amusementparks|golf|iceskating|kayaking|sporttickets|sports|surfing|cinema|zoos');
  const relaxing_request_triposo = getTriposoPOIFromLocation(location_id,'beaches|camping|wildlife|fishing|relaxinapark');
  const eat_request_triposo = getTriposoPOIFromLocation(location_id,'eatingout|breakfast|coffeeandcake|lunch|dinner');
  const shop_request_triposo = getTriposoPOIFromLocation(location_id,'do|shopping');
  const nightlife_request_triposo = getTriposoPOIFromLocation(location_id,'nightlife|comedy|drinks|dancing|pubcrawl|redlight|musicandshows|celebrations|foodexperiences|breweries|showstheatresandmusic');

  const city_request = getTriposoCity(location_id);

  const pois = await Promise.all([
    city_request,
    sightseeing_request_triposo,
    discovering_request_triposo,
    playing_request_triposo,
    relaxing_request_triposo,
    eat_request_triposo,
    shop_request_triposo,
    nightlife_request_triposo,
  ]);
  const [city, see, discover, playing, relax,eat,shop,nightlife] = pois;
  return {
    city: city.results,
    see : see.results,
    discover: discover.results,
    playing: playing.results, 
    relax: relax.results,
    shop: shop.results,
    nightlife: nightlife.results,
    eat:eat.results

  }
}

export const getCity = async (req: Request, res: Response) => {
  const id = req.params.city_id;

  const triposoData = await getCityTriposo(id);
  let see = triposoPlacesToInternal(triposoData.see);
  let eat = triposoPlacesToInternal(triposoData.eat);
  let relax = triposoPlacesToInternal(triposoData.relax);
  let shop = triposoPlacesToInternal(triposoData.shop);
  let play = triposoPlacesToInternal(triposoData.playing);
  let nightlife = triposoPlacesToInternal(triposoData.nightlife);
  let discover = triposoPlacesToInternal(triposoData.discover);
  let city = triposoPlacesToInternal(triposoData.city)[0];

  const see_locations = triposoPlacesToLocations(triposoData.see);
  const discover_locations = triposoPlacesToLocations(triposoData.discover);
  const play_locations = triposoPlacesToLocations(triposoData.playing);
  const eat_locations = triposoPlacesToLocations(triposoData.eat);
  const shop_locations = triposoPlacesToLocations(triposoData.shop);
  const nightlife_locations = triposoPlacesToLocations(triposoData.nightlife);
  const relax_locations = triposoPlacesToLocations(triposoData.relax);


  // displaying top 10 countries in the ui.  I did this to avoid having to make the same call twice.

  res.send({
    city,
    see,
    eat,
    relax,
    shop,
    play,
    nightlife,
    discover,
    see_locations,
    discover_locations,
    play_locations,
    eat_locations,
    shop_locations,
    nightlife_locations,
    relax_locations
  });
};
