export interface SygicPlace {
  id: string; // 'poi:24922'
  name: string; // 'Old Havana',
  name_suffix: string; // 'Havana, Cuba',
  url: string; //'https://go.sygic.com/travel/place?id=poi:24922',
  marker: string; // 'destination:borough',
  categories: string; // [ 'sightseeing', 'discovering' ],
  parent_ids: string[]; //[ 'region:69349', 'city:306', 'region:39455', 'region:2021320', 'country:51', 'continent:7' ],
  perex: string; // 'Old Havana is the city-center and one of the 15 municipalities forming Havana, Cuba.',
  thumbnail_url: string; // 'https://media-cdn.sygictraveldata.com/media/poi:24922' }
}
