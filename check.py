import json, glob, os

errs = []
for f in glob.glob('data/talents/leader/*.json') + glob.glob('data/talents/hunter/*.json'):
    try:
        with open(f, encoding='utf-8-sig') as file:
            d = json.load(file)
            base = os.path.basename(f)
            if 'id' not in d:
                errs.append('Missing id in ' + base)
            else:
                if str(d['id']).lower() != base.replace('.json', '').lower():
                    errs.append('ID mismatch in ' + base + ': ' + str(d['id']))
            if base in ['leader.json', 'hunter.json']:
                print('Parent ' + base + ' paths: ' + str(d.get('paths')))
    except Exception as e:
        errs.append('Error reading ' + f + ': ' + str(e))

for e in errs: print(e)
