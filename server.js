const fs = require("fs");
const app = require("express")();
const express = require("express");
const path = require("path");
const https = require("https");
const http = require("http");
const googleStorage = require("@google-cloud/storage");
const multer = require("multer");
const bodyParser = require("body-parser");
const format = require("util").format;
const request = require("request-promise");
const Vibrant = require("node-vibrant");
const admin = require("firebase-admin");

const serviceAccount = require("./serviceAccountKey.json");
const settings = { /* your settings... */ timestampsInSnapshots: true };
const citizenCode = "US";

admin.initializeApp({
  credential: admin.credential.cert(serviceAccount)
});

const passport_blankpages_map = {
  NOT_REQUIRED: "You do not need to have any blank pages in your passport.",
  ONE: "You need at least one blank page in your passport.",
  ONE_PER_ENTRY: "You need one blank page per entry.",
  SPACE_FOR_STAMP: "You need space for your passport to be stamped.",
  TWO: "You need two blank pages in your passport.",
  TWO_CONSECUTIVE_PER_ENTRY:
    "You need two consecutive blank pages in your passport",
  TWO_PER_ENTRY: "You need two blank pages per entry"
};

const passport_validity_map = {
  DURATION_OF_STAY:
    "Your passport must be valid for the duration of your stay in this country.",
  ONE_MONTH_AFTER_ENTRY:
    "Your passport must be valid for one month after entering this counrty.",
  SIX_MONTHS_AFTER_DURATION_OF_STAY:
    "Your passport must be valid on entry and for six months after the duration of your stay in this country.",
  SIX_MONTHS_AFTER_ENTRY:
    "Your passport must be valid on entry and six months after the date of enrty.",
  SIX_MONTHS_AT_ENTRY:
    "Your passport must be valid for at least six months before entering this country.",
  THREE_MONTHS_AFTER_DURATION_OF_STAY:
    "Your passport must be valid on entry and for three months after the duration of your stay in this country",
  THREE_MONTHS_AFTER_ENTRY:
    "Your passport must be valid on entry and for three months after entering this country",
  VALID_AT_ENTRY: "Your passport must be valid on entry",
  THREE_MONTHS_AFTER_DEPARTURE:
    "Your passport must be valid on entry and three months after your departure date.",
  SIX_MONTHS_AFTER_DEPARTURE:
    "Your passport must be valid on entry and six months after your departure date."
};

const countries_with_states = {
  Argentina: true,
  Australia: true,
  Austria: true,
  Belgium: true,
  "Bosnia and Herzegovina": true,
  Brazil: true,
  Canada: true,
  Comoros: true,
  Ethiopia: true,
  Germany: true,
  India: true,
  Iraq: true,
  Malaysia: true,
  Mexico: true,
  Micronesia: true,
  Nepal: true,
  Nigeria: true,
  Pakistan: true,
  Russia: true,
  "Saint Kitts": true,
  Somalia: true,
  "South Sudan": true,
  Sudan: true,
  Switzerland: true,
  "United Arab Emirates": true,
  "United States of America": true,
  Venezuela: true
};

var db = admin.firestore();
db.settings(settings);
const country_codes = db.collection("countries_code");
const SHERPA_URL = "https://api.joinsherpa.com/v2/entry-requirements/",
  username = "VDLQLCbMmugvsOEtihQ9kfc6nQoeGd",
  password = "nIXaxALFPV0IiwNOvBEBrDCNSw3SCv67R4UEvD9r",
  SHERPA_AUTH =
    "Basic " + new Buffer(username + ":" + password).toString("base64");

app.use(function(req, res, next) {
  res.header("Access-Control-Allow-Origin", "*");
  res.header(
    "Access-Control-Allow-Methods",
    "GET,HEAD,OPTIONS,POST,PUT,DELETE"
  );
  res.header(
    "Access-Control-Allow-Headers",
    "Origin, X-Requested-With, Content-Type, Accept, Authorization, Cache-Control"
  );
  next();
});

app.use(bodyParser.json());

const PROD_MODE = process.argv[2];
const API_KEY = "6SdxevLXN2aviv5g67sac2aySsawGYvJ6UcTmvWE";
let server = http.createServer(app);
if (PROD_MODE) {
  const hskey = fs.readFileSync("/etc/letsencrypt/live/ajibade.me/privkey.pem");
  const hscert = fs.readFileSync(
    "/etc/letsencrypt/live/ajibade.me/fullchain.pem"
  );
  const hschain = fs.readFileSync("/etc/letsencrypt/live/ajibade.me/chain.pem");
  const options = {
    key: hskey,
    cert: hscert,
    ca: hschain
  };
  server = https.createServer(options, app);
}

//Explore continent
app.get("/api/explore/continent/:continent_id", (req, res) => {
  const whatToSee = request({
    method: "GET",
    uri: `https://api.sygictravelapi.com/1.1/en/places/list?&level=poi&categories=sightseeing&parents=continent:${
      req.params.continent_id
    }&limit=10`,
    json: true,
    headers: { "x-api-key": API_KEY }
  });

  const getPopularCities = request({
    method: "GET",
    uri: `https://api.sygictravelapi.com/1.1/en/places/list?rating=.0005:&level=city&parents=continent:${
      req.params.continent_id
    }&limit=10`,
    json: true,
    headers: { "x-api-key": API_KEY }
  });

  const getAllCountries = request({
    method: "GET",
    uri: `https://api.sygictravelapi.com/1.1/en/places/list?level=country&parents=continent:${
      req.params.continent_id
    }&limit=60`,
    json: true,
    headers: { "x-api-key": API_KEY }
  });

  Promise.all([whatToSee, getPopularCities, getAllCountries])
    .then(responses => {
      let points_of_interest = responses[0].data.places.reduce((acc, curr) => {
        return [
          ...acc,
          {
            sygic_id: curr.id,
            image: curr.thumbnail_url,
            name: curr.name,
            name_suffix: curr.name_suffix,
            parent_ids: curr.parent_ids,
            description: curr.perex
          }
        ];
      }, []);

      let popular_cities = responses[1].data.places.reduce((acc, curr) => {
        return [
          ...acc,
          {
            sygic_id: curr.id,
            image: curr.thumbnail_url,
            name: curr.name,
            name_suffix: curr.name_suffix,
            parent_ids: curr.parent_ids,
            description: curr.perex
          }
        ];
      }, []);

      let all_countries = responses[2].data.places.reduce((acc, curr) => {
        return [
          ...acc,
          {
            sygic_id: curr.id,
            image: curr.thumbnail_url,
            name: curr.name,
            name_suffix: curr.name_suffix,
            parent_ids: curr.parent_ids,
            description: curr.perex
          }
        ];
      }, []);
      let popular_countries = all_countries.filter((country, index) => {
        return index < 10;
      });

      res.send({
        popular_countries,
        popular_cities,
        points_of_interest,
        all_countries
      });
    })
    .catch(err => {
      console.log(err);
      res.send(err);
    });
});

//Explore country
app.get("/api/explore/countries/:country_id", (req, res) => {
  /*fs.readFile('./country_23.json', 'utf8', function (err, data) {
    if (err) throw err;
    res.send(JSON.parse(data));
  });*/
  const popular_destinations = request({
    method: "GET",
    uri: encodeURI(
      `https://api.sygictravelapi.com/1.1/en/places/list?&level=region|city|town|island|&parents=${
        req.params.country_id
      }&limit=20`
    ),
    json: true,
    headers: { "x-api-key": API_KEY }
  });

  const sightseeing = request({
    method: "GET",
    uri: `https://api.sygictravelapi.com/1.1/en/places/list?level=poi&categories=sightseeing&parents=${
      req.params.country_id
    }&limit=20`,
    json: true,
    headers: { "x-api-key": API_KEY }
  });

  const discovering = request({
    method: "GET",
    uri: `https://api.sygictravelapi.com/1.1/en/places/list?level=poi&categories=discovering&parents=${
      req.params.country_id
    }&limit=20`,
    json: true,
    headers: { "x-api-key": API_KEY }
  });

  const playing = request({
    method: "GET",
    uri: `https://api.sygictravelapi.com/1.1/en/places/list?level=poi&categories=playing&parents=${
      req.params.country_id
    }&limit=20`,
    json: true,
    headers: { "x-api-key": API_KEY }
  });

  const relaxing = request({
    method: "GET",
    uri: `https://api.sygictravelapi.com/1.1/en/places/list?level=poi&categories=relaxing&parents=${
      req.params.country_id
    }&limit=20`,
    json: true,
    headers: { "x-api-key": API_KEY }
  });

  const country = request({
    method: "GET",
    uri: encodeURI(
      `https://api.sygictravelapi.com/1.1/en/places/${req.params.country_id}`
    ),
    json: true,
    headers: { "x-api-key": API_KEY }
  });

  Promise.all([
    country,
    popular_destinations,
    sightseeing,
    discovering,
    playing,
    relaxing
  ])
    .then(([parent, destinations, sights, discover, play, relax]) => {
      const country_images = parent.data.place.main_media.media.reduce(
        (acc, curr) => {
          return [
            ...acc,
            {
              url: `${curr.url_template.replace(
                /\{[^\]]*?\}/g,
                "1200x800"
              )}.jpg`,
              title: curr.attribution.title,
              author: curr.attribution.author,
              other: curr.attribution.other
            }
          ];
        },
        []
      );

      let country = {
        sygic_id: parent.data.place.id,
        image_usage: parent.data.place.main_media.usage,
        image: parent.data.place.main_media.media[0].url,
        country_images,
        image_template: `${parent.data.place.main_media.media[0].url_template.replace(
          /\{[^\]]*?\}/g,
          "1200x800"
        )}.jpg`,
        name: parent.data.place.name,
        description: parent.data.place.perex
      };

      let sightseeing = sights["data"].places.reduce((acc, curr) => {
        return [
          ...acc,
          {
            sygic_id: curr.id,
            image: curr.thumbnail_url,
            name: curr.name,
            name_suffix: curr.name_suffix,
            parent_ids: curr.parent_ids,
            description: curr.perex
          }
        ];
      }, []);

      let discovering = discover["data"].places.reduce((acc, curr) => {
        return [
          ...acc,
          {
            sygic_id: curr.id,
            image: curr.thumbnail_url,
            name: curr.name,
            name_suffix: curr.name_suffix,
            parent_ids: curr.parent_ids,
            description: curr.perex
          }
        ];
      }, []);

      let playing = play["data"].places.reduce((acc, curr) => {
        return [
          ...acc,
          {
            sygic_id: curr.id,
            image: curr.thumbnail_url,
            name: curr.name,
            name_suffix: curr.name_suffix,
            parent_ids: curr.parent_ids,
            description: curr.perex
          }
        ];
      }, []);

      let relaxing = relax["data"].places.reduce((acc, curr) => {
        return [
          ...acc,
          {
            sygic_id: curr.id,
            image: curr.thumbnail_url,
            name: curr.name,
            name_suffix: curr.name_suffix,
            parent_ids: curr.parent_ids,
            description: curr.perex
          }
        ];
      }, []);

      let popular_destinations = destinations["data"].places.reduce(
        (acc, curr) => {
          return [
            ...acc,
            {
              sygic_id: curr.id,
              image: curr.thumbnail_url,
              name: curr.name,
              name_suffix: curr.name_suffix,
              parent_ids: curr.parent_ids,
              description: curr.perex
            }
          ];
        },
        []
      );

      let country_color = Vibrant.from(country.image).getPalette();
      let country_name = country.name;
      if (country_name == "United States of America") {
        country_name = "United States";
      }
      let country_code = country_codes.doc(country_name).get();
      const data = {
        country,
        popular_destinations,
        sightseeing,
        discovering,
        playing,
        relaxing
      };

      if (countries_with_states[country.name]) {
        console.log(
          `https://api.sygictravelapi.com/1.1/en/places/list?parents=${
            req.params.country_id
          }&level=state&limit=100`
        );
        const states = request({
          method: "GET",
          uri: encodeURI(
            `https://api.sygictravelapi.com/1.1/en/places/list?parents=${
              req.params.country_id
            }&level=state&limit=100`
          ),
          json: true,
          headers: { "x-api-key": API_KEY }
        });
        return Promise.all([data, country_color, country_code, states]);
      }

      return Promise.all([data, country_color, country_code]);

      //res.send({popular_destinations, points_of_interest, popular_tours});
    })
    .then(([data, color, country_code, states]) => {
      let popular_destinations = data.popular_destinations;
      let sightseeing = data.sightseeing;
      let discover = data.discovering;
      let play = data.playing;
      let relax = data.relaxing;
      let country = data.country;
      country.color = `rgb(${color.Vibrant._rgb[0]},${color.Vibrant._rgb[1]},${
        color.Vibrant._rgb[2]
      })`;
      let visaData = country_code.data();
      let visaCode = visaData.abbreviation;

      if (visaData && visaCode !== citizenCode) {
        var visa = request({
          method: "GET",
          uri: `${SHERPA_URL}${citizenCode}-${visaCode}`,
          json: true,
          headers: { Authorization: SHERPA_AUTH }
        });
      }

      country.visa = null;

      let noVisaData = {
        country,
        popular_destinations,
        sightseeing,
        discover,
        play,
        relax
      };
      noVisaData.states = [];

      if (states) {
        let top_states = states["data"].places.reduce((acc, curr) => {
          return [
            ...acc,
            {
              sygic_id: curr.id,
              image: curr.thumbnail_url,
              name: curr.name,
              name_suffix: curr.name_suffix,
              parent_ids: curr.parent_ids,
              description: curr.perex
            }
          ];
        }, []);
        noVisaData.states = top_states;
      }

      if (!country_code.exists || visaCode == citizenCode) {
        return noVisaData;
      }

      return Promise.all([noVisaData, visa]);
    })
    .then(data => {
      if (data instanceof Array) {
        let {
          country,
          popular_destinations,
          sightseeing,
          discover,
          play,
          relax,
          states
        } = data[0];
        let visa = data[1];
        visa.visa = data[1].visa ? data[1].visa[0] : null;
        visa.passport = visa.passport
          ? visa.passport
          : { passport_validity: null, blank_pages: null };
        let passport_valid = visa.passport.passport_validity
          ? visa.passport.passport_validity
          : null;
        let blank_pages = visa.passport.blank_pages
          ? visa.passport.blank_pages
          : null;
        visa.passport.passport_validity =
          passport_valid && passport_validity_map[passport_valid]
            ? passport_validity_map[passport_valid]
            : `${
                passport_validity_map["VALID_AT_ENTRY"]
              } Make sure to check for additional requirements.`;

        visa.passport.blank_pages =
          blank_pages && passport_blankpages_map[blank_pages]
            ? passport_blankpages_map[blank_pages]
            : "To be safe make sure to have at least one blank page in your passport.";

        country.visa = visa;
        res.send({
          country,
          popular_destinations,
          sightseeing,
          discover,
          play,
          relax,
          states
        });
      } else {
        res.send(data);
      }
    })
    .catch(err => {
      console.log(err);
      res.send(err);
    });
});

//City
app.get("/api/explore/cities/:city_id", (req, res) => {});
app.get("/api/explore/cities/:city_id/sightseeing", (req, res) => {});

const uploadImageToStorage = file => {
  let prom = new Promise((resolve, reject) => {
    if (!file) {
      reject("No image file");
    }
    let newFileName = `${Date.now()}_${file.originalname}`;

    let fileUpload = bucket.file(newFileName);

    const blobStream = fileUpload.createWriteStream({
      metadata: {
        contentType: file.mimetype
      }
    });

    blobStream.on("error", error => {
      reject("Something is wrong! Unable to upload at the moment.");
    });

    blobStream.on("finish", e => {
      // The public URL can be used to directly access the file via HTTP.
      const url = format(
        `https://firebasestorage.googleapis.com/v0/b/${
          bucket.name
        }/o/${encodeURIComponent(fileUpload.name)}?alt=media`
      );
      resolve({ url, newFileName });
    });

    blobStream.end(file.buffer);
  });
  return prom;
};

const listen = server.listen(3002, function() {
  var host = server.address().address;
  var port = server.address().port;
  console.log("app listening at //%s:%s", host, port);
});

listen.timeout = 900000;
