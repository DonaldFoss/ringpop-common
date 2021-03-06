// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// hashing algorithm might change upon ringpop implementation
var _ = require('lodash');
var farmhash = require('farmhash');

function generateChecksumString(members) {
    return _
        .chain(members)
        .filter(function (member) {
            // remove members that are in the tombstone state
            return ['tombstone'].indexOf(member.status) < 0;
        })
        .map(function (member) {
            var address = member.address || (member.host + ':' + member.port);

            var labelChecksum = 0;

            if (member.labels) {
                Object.keys(member.labels).forEach(function (key) {
                    var value = member.labels[key];

                    var keyLength = Buffer.byteLength(key);
                    var valueLength = Buffer.byteLength(value);

                    var pos = 0;
                    var buf = new Buffer(8 + keyLength + valueLength);
                    buf.writeInt32BE(keyLength, pos);
                    pos += 4
                    pos += buf.write(key, pos);
                    buf.writeInt32BE(valueLength, pos);
                    pos += 4
                    pos += buf.write(value, pos);

                    labelChecksum ^= farmhash.fingerprint32(buf);
                });
            }

            // concat all parts of the checksum string
            var checksumString = '';
            checksumString += address;
            checksumString += member.status;
            checksumString += member.incarnationNumber;

            if (labelChecksum != 0) {
                // we only add the label checksum to the string when it is not 0
                // This guarantees that the change is backwards compatible and
                // ringpop with label support can correctly synchronize with a
                // version that has no labels as long as there are no labels set
                // on ringpop.
                checksumString += '#labels' + labelChecksum;
            }

            return checksumString;
        })
        .value()
        .sort()
        .join(';');
}
// entries must have address (hostport), status (e.g. "alive"), and incarnation numbers
function checksumGo(members) {
    //add extra semi-colon to be compatible with the go implementation
    var checksumString = generateChecksumString(members) + ';';
    return farmhash.fingerprint32(checksumString);
}

function checksumNode(members) {
    var checksumString = generateChecksumString(members);
    return farmhash.hash32(checksumString);
}

function checksumNodeCrossPlatform(members) {
    var checksumString = generateChecksumString(members);
    return farmhash.fingerprint32(checksumString);
}

function detect(members, checksum) {
    if(checksumGo(members) === checksum) {
        return checksumGo;
    } else if(checksumNode(members) === checksum) {
        return checksumNode;
    } else if (checksumNodeCrossPlatform(members) === checksum) {
        return checksumNodeCrossPlatform;
    } else {
        throw new Error('checksum method undetectable!');
    }
}

module.exports = {
    detect: detect
};
