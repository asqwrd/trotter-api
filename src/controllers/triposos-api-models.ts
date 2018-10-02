export interface TriposoPlace {
  id: string; // 'poi:24922'
  location_id: string; // 'poi:24922'
  name: string; // 'Old Havana',
  opening_hours: string | null;
  coordinates: {
    latitude: number
    longitude: number
  };
  description: {text:string};
  content:{
    sections: {
      body:string,
    }[]
  };
  images:{
    owner_url: string,
    sizes: {
      medium:{
        url:string;
      }
    }
  }[];
  facebook_id: string;
  foursquare_id: string;
  google_place_id: string;
  snippet:string;
}

export interface PlacesData {
  data: { places: TriposoPlace[] };
}

export interface PlaceTriposo {
  id: string; // 'poi:24922'
  location_id: string; // 'poi:24922'
  name: string; // 'Old Havana',
  opening_hours: string | null;
  coordinates: {
    latitude: number
    longitude: number
  };
  description: {text:string};
  content:{
    sections: {
      body:string,
    }[]
  };
  images:{
    owner_url: string,
    sizes: {
      medium:{
        url:string;
      }
    }
  }[];
  facebook_id: string;
  foursquare_id: string;
  google_place_id: string;
  snippet:string;

}

