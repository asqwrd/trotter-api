const fs = require('fs');
const app = require('express')();
const express = require('express');
const path = require('path');
const https = require('https');
const http = require('http');
const firebase = require('firebase');
const googleStorage = require('@google-cloud/storage');
const multer = require('multer');
const bodyParser = require('body-parser')
const format = require('util').format;

app.use(function (req, res, next) {
    res.header("Access-Control-Allow-Origin", "*");
    res.header("Access-Control-Allow-Methods", "GET,HEAD,OPTIONS,POST,PUT,DELETE");
    res.header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization, Cache-Control");
    next();
});

app.use(bodyParser.json());
app.use(express.static(path.join(__dirname, 'build')));

/*

const storage = googleStorage({
    projectId: "eric-wedding",
    keyFilename: "serviceAccountKey.json"
});
const bucket = storage.bucket("eric-wedding.appspot.com");

const storage_mul = multer({
    storage: multer.memoryStorage(),
    limits: {
        fileSize: 10 * 1024 * 1024 // no larger than 5mb, you can change as needed.
    }
});



// Initialize Firebase
const config = {
    apiKey: "AIzaSyDuQZlorNPGMgFgAn8HKWzgZ57RxrUW72U",
    authDomain: "eric-wedding.firebaseapp.com",
    databaseURL: "https://eric-wedding.firebaseio.com",
    projectId: "eric-wedding",
    storageBucket: "eric-wedding.appspot.com",
    messagingSenderId: "66360165093"
};
firebase.initializeApp(config);
const db = firebase.database().ref();
const slider_images = db.child('images');
const home_images = slider_images.child('home').orderByKey();
const photos_images = slider_images.child('photos').orderByKey();
const rsvpDB = db.child('rsvp');
const codeDb = db.child('codes');
//codeDb.push().set({value:1234,used:false});
const generateCodes = (num) => {
    let codes = {};
    let codesCount = 0;
    while (codesCount < num) {
        let code = Math.random().toString(36).slice(-8);
        if (!codes[code]) {
            codes[code] = code;
            codeDb.push().set({ code: code, used: false });
            codesCount++;
        }
    }
}

//generateCodes(100);
*/

const PROD_MODE = process.argv[2];
let server = http.createServer(app);
if (PROD_MODE) {
    const hskey = fs.readFileSync('/etc/letsencrypt/live/ajibade.me/privkey.pem');
    const hscert = fs.readFileSync('/etc/letsencrypt/live/ajibade.me/fullchain.pem');
    const hschain = fs.readFileSync('/etc/letsencrypt/live/ajibade.me/chain.pem');
    const options = {
        key: hskey,
        cert: hscert,
        ca: hschain
    };
    server = https.createServer(options, app);

}


app.get('/api/countries/:continent_id', (req, res) => {
  
});

//Explore continent
app.get("/api/explore/continent/:continent_id/popular_countries", (req, res) => {});
app.get("/api/explore/continent/:continent_id/popular_sightseeing", (req, res) => {});


app.get("/api/explore/continent/:continent_id/popular_cities", (req, res) => {});
app.get("/api/explore/continent/:continent_id/popular_sightseeing", (req, res) => {});

//Explore country
app.get("/api/explore/countries/:country_id/popular_cities", (req, res) => { });
app.get("/api/explore/countries/:country_id/popular_sightseeing", (req, res) => { });

//City
app.get("/api/explore/cities/:city_id", (req, res) => {});
app.get("/api/explore/cities/:city_id/sightseeing", (req, res) => {});

const uploadImageToStorage = (file) => {
    let prom = new Promise((resolve, reject) => {
        if (!file) {
            reject('No image file');
        }
        let newFileName = `${Date.now()}_${file.originalname}`;

        let fileUpload = bucket.file(newFileName);

        const blobStream = fileUpload.createWriteStream({
            metadata: {
                contentType: file.mimetype
            }
        });

        blobStream.on('error', (error) => {
            reject('Something is wrong! Unable to upload at the moment.');
        });

        blobStream.on('finish', (e) => {
            // The public URL can be used to directly access the file via HTTP.
            const url = format(`https://firebasestorage.googleapis.com/v0/b/${bucket.name}/o/${encodeURIComponent(fileUpload.name)}?alt=media`);
            resolve({ url, newFileName });
        });

        blobStream.end(file.buffer);
    });
    return prom;
}

const listen = server.listen(3002, function () {
    var host = server.address().address;
    var port = server.address().port;
    console.log('app listening at //%s:%s', host, port);
});

listen.timeout = 900000;