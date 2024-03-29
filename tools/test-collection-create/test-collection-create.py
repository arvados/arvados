#!/usr/bin/env python3
#
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

import argparse
import logging
import random
import string
import sys

import arvados
import arvados.collection

logger = logging.getLogger('arvados.test_collection_create')
logger.setLevel(logging.INFO)

max_manifest_size = 127*1024*1024

opts = argparse.ArgumentParser(add_help=False)
opts.add_argument('--min-files', type=int, default=30000, help="""
Minimum number of files on each directory. Default: 30000.
""")
opts.add_argument('--max-files', type=int, default=30000, help="""
Maximum number of files on each directory. Default: 30000.
""")
opts.add_argument('--min-depth', type=int, default=0, help="""
Minimum depth for the created tree structure. Default: 0.
""")
opts.add_argument('--max-depth', type=int, default=0, help="""
Maximum depth for the created tree structure. Default: 0.
""")
opts.add_argument('--min-subdirs', type=int, default=1, help="""
Minimum number of subdirectories created at every depth level. Default: 1.
""")
opts.add_argument('--max-subdirs', type=int, default=10, help="""
Maximum number of subdirectories created at every depth level. Default: 10.
""")
opts.add_argument('--debug', action='store_true', default=False, help="""
Sets logging level to DEBUG.
""")

arg_parser = argparse.ArgumentParser(
    description='Create a collection with garbage data for testing purposes.',
    parents=[opts])

adjectives = ['abandoned','able','absolute','adorable','adventurous','academic',
    'acceptable','acclaimed','accomplished','accurate','aching','acidic','acrobatic',
    'active','actual','adept','admirable','admired','adolescent','adorable','adored',
    'advanced','afraid','affectionate','aged','aggravating','aggressive','agile',
    'agitated','agonizing','agreeable','ajar','alarmed','alarming','alert','alienated',
    'alive','all','altruistic','amazing','ambitious','ample','amused','amusing','anchored',
    'ancient','angelic','angry','anguished','animated','annual','another','antique',
    'anxious','any','apprehensive','appropriate','apt','arctic','arid','aromatic','artistic',
    'ashamed','assured','astonishing','athletic','attached','attentive','attractive',
    'austere','authentic','authorized','automatic','avaricious','average','aware','awesome',
    'awful','awkward','babyish','bad','back','baggy','bare','barren','basic','beautiful',
    'belated','beloved','beneficial','better','best','bewitched','big','big-hearted',
    'biodegradable','bite-sized','bitter','black','black-and-white','bland','blank',
    'blaring','bleak','blind','blissful','blond','blue','blushing','bogus','boiling',
    'bold','bony','boring','bossy','both','bouncy','bountiful','bowed','brave','breakable',
    'brief','bright','brilliant','brisk','broken','bronze','brown','bruised','bubbly',
    'bulky','bumpy','buoyant','burdensome','burly','bustling','busy','buttery','buzzing',
    'calculating','calm','candid','canine','capital','carefree','careful','careless',
    'caring','cautious','cavernous','celebrated','charming','cheap','cheerful','cheery',
    'chief','chilly','chubby','circular','classic','clean','clear','clear-cut','clever',
    'close','closed','cloudy','clueless','clumsy','cluttered','coarse','cold','colorful',
    'colorless','colossal','comfortable','common','compassionate','competent','complete',
    'complex','complicated','composed','concerned','concrete','confused','conscious',
    'considerate','constant','content','conventional','cooked','cool','cooperative',
    'coordinated','corny','corrupt','costly','courageous','courteous','crafty','crazy',
    'creamy','creative','creepy','criminal','crisp','critical','crooked','crowded',
    'cruel','crushing','cuddly','cultivated','cultured','cumbersome','curly','curvy',
    'cute','cylindrical','damaged','damp','dangerous','dapper','daring','darling','dark',
    'dazzling','dead','deadly','deafening','dear','dearest','decent','decimal','decisive',
    'deep','defenseless','defensive','defiant','deficient','definite','definitive','delayed',
    'delectable','delicious','delightful','delirious','demanding','dense','dental',
    'dependable','dependent','descriptive','deserted','detailed','determined','devoted',
    'different','difficult','digital','diligent','dim','dimpled','dimwitted','direct',
    'disastrous','discrete','disfigured','disgusting','disloyal','dismal','distant',
    'downright','dreary','dirty','disguised','dishonest','dismal','distant','distinct',
    'distorted','dizzy','dopey','doting','double','downright','drab','drafty','dramatic',
    'dreary','droopy','dry','dual','dull','dutiful','each','eager','earnest','early',
    'easy','easy-going','ecstatic','edible','educated','elaborate','elastic','elated',
    'elderly','electric','elegant','elementary','elliptical','embarrassed','embellished',
    'eminent','emotional','empty','enchanted','enchanting','energetic','enlightened',
    'enormous','enraged','entire','envious','equal','equatorial','essential','esteemed',
    'ethical','euphoric','even','evergreen','everlasting','every','evil','exalted',
    'excellent','exemplary','exhausted','excitable','excited','exciting','exotic',
    'expensive','experienced','expert','extraneous','extroverted','extra-large','extra-small',
    'fabulous','failing','faint','fair','faithful','fake','false','familiar','famous',
    'fancy','fantastic','far','faraway','far-flung','far-off','fast','fat','fatal',
    'fatherly','favorable','favorite','fearful','fearless','feisty','feline','female',
    'feminine','few','fickle','filthy','fine','finished','firm','first','firsthand',
    'fitting','fixed','flaky','flamboyant','flashy','flat','flawed','flawless','flickering',
    'flimsy','flippant','flowery','fluffy','fluid','flustered','focused','fond','foolhardy',
    'foolish','forceful','forked','formal','forsaken','forthright','fortunate','fragrant',
    'frail','frank','frayed','free','French','fresh','frequent','friendly','frightened',
    'frightening','frigid','frilly','frizzy','frivolous','front','frosty','frozen',
    'frugal','fruitful','full','fumbling','functional','funny','fussy','fuzzy','gargantuan',
    'gaseous','general','generous','gentle','genuine','giant','giddy','gigantic','gifted',
    'giving','glamorous','glaring','glass','gleaming','gleeful','glistening','glittering',
    'gloomy','glorious','glossy','glum','golden','good','good-natured','gorgeous',
    'graceful','gracious','grand','grandiose','granular','grateful','grave','gray',
    'great','greedy','green','gregarious','grim','grimy','gripping','grizzled','gross',
    'grotesque','grouchy','grounded','growing','growling','grown','grubby','gruesome',
    'grumpy','guilty','gullible','gummy','hairy','half','handmade','handsome','handy',
    'happy','happy-go-lucky','hard','hard-to-find','harmful','harmless','harmonious',
    'harsh','hasty','hateful','haunting','healthy','heartfelt','hearty','heavenly',
    'heavy','hefty','helpful','helpless','hidden','hideous','high','high-level','hilarious',
    'hoarse','hollow','homely','honest','honorable','honored','hopeful','horrible',
    'hospitable','hot','huge','humble','humiliating','humming','humongous','hungry',
    'hurtful','husky','icky','icy','ideal','idealistic','identical','idle','idiotic',
    'idolized','ignorant','ill','illegal','ill-fated','ill-informed','illiterate',
    'illustrious','imaginary','imaginative','immaculate','immaterial','immediate',
    'immense','impassioned','impeccable','impartial','imperfect','imperturbable','impish',
    'impolite','important','impossible','impractical','impressionable','impressive',
    'improbable','impure','inborn','incomparable','incompatible','incomplete','inconsequential',
    'incredible','indelible','inexperienced','indolent','infamous','infantile','infatuated',
    'inferior','infinite','informal','innocent','insecure','insidious','insignificant',
    'insistent','instructive','insubstantial','intelligent','intent','intentional',
    'interesting','internal','international','intrepid','ironclad','irresponsible',
    'irritating','itchy','jaded','jagged','jam-packed','jaunty','jealous','jittery',
    'joint','jolly','jovial','joyful','joyous','jubilant','judicious','juicy','jumbo',
    'junior','jumpy','juvenile','kaleidoscopic','keen','key','kind','kindhearted','kindly',
    'klutzy','knobby','knotty','knowledgeable','knowing','known','kooky','kosher','lame',
    'lanky','large','last','lasting','late','lavish','lawful','lazy','leading','lean',
    'leafy','left','legal','legitimate','light','lighthearted','likable','likely','limited',
    'limp','limping','linear','lined','liquid','little','live','lively','livid','loathsome',
    'lone','lonely','long','long-term','loose','lopsided','lost','loud','lovable','lovely',
    'loving','low','loyal','lucky','lumbering','luminous','lumpy','lustrous','luxurious',
    'mad','made-up','magnificent','majestic','major','male','mammoth','married','marvelous',
    'masculine','massive','mature','meager','mealy','mean','measly','meaty','medical',
    'mediocre','medium','meek','mellow','melodic','memorable','menacing','merry','messy',
    'metallic','mild','milky','mindless','miniature','minor','minty','miserable','miserly',
    'misguided','misty','mixed','modern','modest','moist','monstrous','monthly','monumental',
    'moral','mortified','motherly','motionless','mountainous','muddy','muffled','multicolored',
    'mundane','murky','mushy','musty','muted','mysterious','naive','narrow','nasty','natural',
    'naughty','nautical','near','neat','necessary','needy','negative','neglected','negligible',
    'neighboring','nervous','new','next','nice','nifty','nimble','nippy','nocturnal','noisy',
    'nonstop','normal','notable','noted','noteworthy','novel','noxious','numb','nutritious',
    'nutty','obedient','obese','oblong','oily','oblong','obvious','occasional','odd',
    'oddball','offbeat','offensive','official','old','old-fashioned','only','open','optimal',
    'optimistic','opulent','orange','orderly','organic','ornate','ornery','ordinary',
    'original','other','our','outlying','outgoing','outlandish','outrageous','outstanding',
    'oval','overcooked','overdue','overjoyed','overlooked','palatable','pale','paltry',
    'parallel','parched','partial','passionate','past','pastel','peaceful','peppery',
    'perfect','perfumed','periodic','perky','personal','pertinent','pesky','pessimistic',
    'petty','phony','physical','piercing','pink','pitiful','plain','plaintive','plastic',
    'playful','pleasant','pleased','pleasing','plump','plush','polished','polite','political',
    'pointed','pointless','poised','poor','popular','portly','posh','positive','possible',
    'potable','powerful','powerless','practical','precious','present','prestigious',
    'pretty','precious','previous','pricey','prickly','primary','prime','pristine','private',
    'prize','probable','productive','profitable','profuse','proper','proud','prudent',
    'punctual','pungent','puny','pure','purple','pushy','putrid','puzzled','puzzling',
    'quaint','qualified','quarrelsome','quarterly','queasy','querulous','questionable',
    'quick','quick-witted','quiet','quintessential','quirky','quixotic','quizzical',
    'radiant','ragged','rapid','rare','rash','raw','recent','reckless','rectangular',
    'ready','real','realistic','reasonable','red','reflecting','regal','regular',
    'reliable','relieved','remarkable','remorseful','remote','repentant','required',
    'respectful','responsible','repulsive','revolving','rewarding','rich','rigid',
    'right','ringed','ripe','roasted','robust','rosy','rotating','rotten','rough',
    'round','rowdy','royal','rubbery','rundown','ruddy','rude','runny','rural','rusty',
    'sad','safe','salty','same','sandy','sane','sarcastic','sardonic','satisfied',
    'scaly','scarce','scared','scary','scented','scholarly','scientific','scornful',
    'scratchy','scrawny','second','secondary','second-hand','secret','self-assured',
    'self-reliant','selfish','sentimental','separate','serene','serious','serpentine',
    'several','severe','shabby','shadowy','shady','shallow','shameful','shameless',
    'sharp','shimmering','shiny','shocked','shocking','shoddy','short','short-term',
    'showy','shrill','shy','sick','silent','silky','silly','silver','similar','simple',
    'simplistic','sinful','single','sizzling','skeletal','skinny','sleepy','slight',
    'slim','slimy','slippery','slow','slushy','small','smart','smoggy','smooth','smug',
    'snappy','snarling','sneaky','sniveling','snoopy','sociable','soft','soggy','solid',
    'somber','some','spherical','sophisticated','sore','sorrowful','soulful','soupy',
    'sour','Spanish','sparkling','sparse','specific','spectacular','speedy','spicy',
    'spiffy','spirited','spiteful','splendid','spotless','spotted','spry','square',
    'squeaky','squiggly','stable','staid','stained','stale','standard','starchy','stark',
    'starry','steep','sticky','stiff','stimulating','stingy','stormy','straight','strange',
    'steel','strict','strident','striking','striped','strong','studious','stunning',
    'stupendous','stupid','sturdy','stylish','subdued','submissive','substantial','subtle',
    'suburban','sudden','sugary','sunny','super','superb','superficial','superior',
    'supportive','sure-footed','surprised','suspicious','svelte','sweaty','sweet','sweltering',
    'swift','sympathetic','tall','talkative','tame','tan','tangible','tart','tasty',
    'tattered','taut','tedious','teeming','tempting','tender','tense','tepid','terrible',
    'terrific','testy','thankful','that','these','thick','thin','third','thirsty','this',
    'thorough','thorny','those','thoughtful','threadbare','thrifty','thunderous','tidy',
    'tight','timely','tinted','tiny','tired','torn','total','tough','traumatic','treasured',
    'tremendous','tragic','trained','tremendous','triangular','tricky','trifling','trim',
    'trivial','troubled','true','trusting','trustworthy','trusty','truthful','tubby',
    'turbulent','twin','ugly','ultimate','unacceptable','unaware','uncomfortable',
    'uncommon','unconscious','understated','unequaled','uneven','unfinished','unfit',
    'unfolded','unfortunate','unhappy','unhealthy','uniform','unimportant','unique',
    'united','unkempt','unknown','unlawful','unlined','unlucky','unnatural','unpleasant',
    'unrealistic','unripe','unruly','unselfish','unsightly','unsteady','unsung','untidy',
    'untimely','untried','untrue','unused','unusual','unwelcome','unwieldy','unwilling',
    'unwitting','unwritten','upbeat','upright','upset','urban','usable','used','useful',
    'useless','utilized','utter','vacant','vague','vain','valid','valuable','vapid',
    'variable','vast','velvety','venerated','vengeful','verifiable','vibrant','vicious',
    'victorious','vigilant','vigorous','villainous','violet','violent','virtual',
    'virtuous','visible','vital','vivacious','vivid','voluminous','wan','warlike','warm',
    'warmhearted','warped','wary','wasteful','watchful','waterlogged','watery','wavy',
    'wealthy','weak','weary','webbed','wee','weekly','weepy','weighty','weird','welcome',
    'well-documented','well-groomed','well-informed','well-lit','well-made','well-off',
    'well-to-do','well-worn','wet','which','whimsical','whirlwind','whispered','white',
    'whole','whopping','wicked','wide','wide-eyed','wiggly','wild','willing','wilted',
    'winding','windy','winged','wiry','wise','witty','wobbly','woeful','wonderful',
    'wooden','woozy','wordy','worldly','worn','worried','worrisome','worse','worst',
    'worthless','worthwhile','worthy','wrathful','wretched','writhing','wrong','wry',
    'yawning','yearly','yellow','yellowish','young','youthful','yummy','zany','zealous',
    'zesty','zigzag']
nouns = ['people','history','way','art','world','information','map','two','family',
    'government','health','system','computer','meat','year','thanks','music','person',
    'reading','method','data','food','understanding','theory','law','bird','literature',
    'problem','software','control','knowledge','power','ability','economics','love',
    'internet','television','science','library','nature','fact','product','idea',
    'temperature','investment','area','society','activity','story','industry','media',
    'thing','oven','community','definition','safety','quality','development','language',
    'management','player','variety','video','week','security','country','exam','movie',
    'organization','equipment','physics','analysis','policy','series','thought','basis',
    'boyfriend','direction','strategy','technology','army','camera','freedom','paper',
    'environment','child','instance','month','truth','marketing','university','writing',
    'article','department','difference','goal','news','audience','fishing','growth',
    'income','marriage','user','combination','failure','meaning','medicine','philosophy',
    'teacher','communication','night','chemistry','disease','disk','energy','nation',
    'road','role','soup','advertising','location','success','addition','apartment','education',
    'math','moment','painting','politics','attention','decision','event','property',
    'shopping','student','wood','competition','distribution','entertainment','office',
    'population','president','unit','category','cigarette','context','introduction',
    'opportunity','performance','driver','flight','length','magazine','newspaper',
    'relationship','teaching','cell','dealer','finding','lake','member','message','phone',
    'scene','appearance','association','concept','customer','death','discussion','housing',
    'inflation','insurance','mood','woman','advice','blood','effort','expression','importance',
    'opinion','payment','reality','responsibility','situation','skill','statement','wealth',
    'application','city','county','depth','estate','foundation','grandmother','heart',
    'perspective','photo','recipe','studio','topic','collection','depression','imagination',
    'passion','percentage','resource','setting','ad','agency','college','connection',
    'criticism','debt','description','memory','patience','secretary','solution','administration',
    'aspect','attitude','director','personality','psychology','recommendation','response',
    'selection','storage','version','alcohol','argument','complaint','contract','emphasis',
    'highway','loss','membership','possession','preparation','steak','union','agreement',
    'cancer','currency','employment','engineering','entry','interaction','mixture','preference',
    'region','republic','tradition','virus','actor','classroom','delivery','device',
    'difficulty','drama','election','engine','football','guidance','hotel','owner',
    'priority','protection','suggestion','tension','variation','anxiety','atmosphere',
    'awareness','bath','bread','candidate','climate','comparison','confusion','construction',
    'elevator','emotion','employee','employer','guest','height','leadership','mall','manager',
    'operation','recording','sample','transportation','charity','cousin','disaster','editor',
    'efficiency','excitement','extent','feedback','guitar','homework','leader','mom','outcome',
    'permission','presentation','promotion','reflection','refrigerator','resolution','revenue',
    'session','singer','tennis','basket','bonus','cabinet','childhood','church','clothes','coffee',
    'dinner','drawing','hair','hearing','initiative','judgment','lab','measurement','mode','mud',
    'orange','poetry','police','possibility','procedure','queen','ratio','relation','restaurant',
    'satisfaction','sector','signature','significance','song','tooth','town','vehicle','volume','wife',
    'accident','airport','appointment','arrival','assumption','baseball','chapter','committee',
    'conversation','database','enthusiasm','error','explanation','farmer','gate','girl','hall',
    'historian','hospital','injury','instruction','maintenance','manufacturer','meal','perception','pie',
    'poem','presence','proposal','reception','replacement','revolution','river','son','speech','tea',
    'village','warning','winner','worker','writer','assistance','breath','buyer','chest','chocolate',
    'conclusion','contribution','cookie','courage','dad','desk','drawer','establishment','examination',
    'garbage','grocery','honey','impression','improvement','independence','insect','inspection',
    'inspector','king','ladder','menu','penalty','piano','potato','profession','professor','quantity',
    'reaction','requirement','salad','sister','supermarket','tongue','weakness','wedding','affair',
    'ambition','analyst','apple','assignment','assistant','bathroom','bedroom','beer','birthday',
    'celebration','championship','cheek','client','consequence','departure','diamond','dirt','ear',
    'fortune','friendship','funeral','gene','girlfriend','hat','indication','intention','lady',
    'midnight','negotiation','obligation','passenger','pizza','platform','poet','pollution',
    'recognition','reputation','shirt','sir','speaker','stranger','surgery','sympathy','tale','throat',
    'trainer','uncle','youth','time','work','film','water','money','example','while','business','study',
    'game','life','form','air','day','place','number','part','field','fish','back','process','heat',
    'hand','experience','job','book','end','point','type','home','economy','value','body','market',
    'guide','interest','state','radio','course','company','price','size','card','list','mind','trade',
    'line','care','group','risk','word','fat','force','key','light','training','name','school','top',
    'amount','level','order','practice','research','sense','service','piece','web','boss','sport','fun',
    'house','page','term','test','answer','sound','focus','matter','kind','soil','board','oil','picture',
    'access','garden','range','rate','reason','future','site','demand','exercise','image','case','cause',
    'coast','action','age','bad','boat','record','result','section','building','mouse','cash','class',
    'nothing','period','plan','store','tax','side','subject','space','rule','stock','weather','chance',
    'figure','man','model','source','beginning','earth','program','chicken','design','feature','head',
    'material','purpose','question','rock','salt','act','birth','car','dog','object','scale','sun',
    'note','profit','rent','speed','style','war','bank','craft','half','inside','outside','standard',
    'bus','exchange','eye','fire','position','pressure','stress','advantage','benefit','box','frame',
    'issue','step','cycle','face','item','metal','paint','review','room','screen','structure','view',
    'account','ball','discipline','medium','share','balance','bit','black','bottom','choice','gift',
    'impact','machine','shape','tool','wind','address','average','career','culture','morning','pot',
    'sign','table','task','condition','contact','credit','egg','hope','ice','network','north','square',
    'attempt','date','effect','link','post','star','voice','capital','challenge','friend','self','shot',
    'brush','couple','debate','exit','front','function','lack','living','plant','plastic','spot',
    'summer','taste','theme','track','wing','brain','button','click','desire','foot','gas','influence',
    'notice','rain','wall','base','damage','distance','feeling','pair','savings','staff','sugar',
    'target','text','animal','author','budget','discount','file','ground','lesson','minute','officer',
    'phase','reference','register','sky','stage','stick','title','trouble','bowl','bridge','campaign',
    'character','club','edge','evidence','fan','letter','lock','maximum','novel','option','pack','park',
    'plenty','quarter','skin','sort','weight','baby','background','carry','dish','factor','fruit',
    'glass','joint','master','muscle','red','strength','traffic','trip','vegetable','appeal','chart',
    'gear','ideal','kitchen','land','log','mother','net','party','principle','relative','sale','season',
    'signal','spirit','street','tree','wave','belt','bench','commission','copy','drop','minimum','path',
    'progress','project','sea','south','status','stuff','ticket','tour','angle','blue','breakfast',
    'confidence','daughter','degree','doctor','dot','dream','duty','essay','father','fee','finance',
    'hour','juice','limit','luck','milk','mouth','peace','pipe','seat','stable','storm','substance',
    'team','trick','afternoon','bat','beach','blank','catch','chain','consideration','cream','crew',
    'detail','gold','interview','kid','mark','match','mission','pain','pleasure','score','screw','sex',
    'shop','shower','suit','tone','window','agent','band','block','bone','calendar','cap','coat',
    'contest','corner','court','cup','district','door','east','finger','garage','guarantee','hole',
    'hook','implement','layer','lecture','lie','manner','meeting','nose','parking','partner','profile',
    'respect','rice','routine','schedule','swimming','telephone','tip','winter','airline','bag','battle',
    'bed','bill','bother','cake','code','curve','designer','dimension','dress','ease','emergency',
    'evening','extension','farm','fight','gap','grade','holiday','horror','horse','host','husband',
    'loan','mistake','mountain','nail','noise','occasion','package','patient','pause','phrase','proof',
    'race','relief','sand','sentence','shoulder','smoke','stomach','string','tourist','towel','vacation',
    'west','wheel','wine','arm','aside','associate','bet','blow','border','branch','breast','brother',
    'buddy','bunch','chip','coach','cross','document','draft','dust','expert','floor','god','golf',
    'habit','iron','judge','knife','landscape','league','mail','mess','native','opening','parent',
    'pattern','pin','pool','pound','request','salary','shame','shelter','shoe','silver','tackle','tank',
    'trust','assist','bake','bar','bell','bike','blame','boy','brick','chair','closet','clue','collar',
    'comment','conference','devil','diet','fear','fuel','glove','jacket','lunch','monitor','mortgage',
    'nurse','pace','panic','peak','plane','reward','row','sandwich','shock','spite','spray','surprise',
    'till','transition','weekend','welcome','yard','alarm','bend','bicycle','bite','blind','bottle',
    'cable','candle','clerk','cloud','concert','counter','flower','grandfather','harm','knee','lawyer',
    'leather','load','mirror','neck','pension','plate','purple','ruin','ship','skirt','slice','snow',
    'specialist','stroke','switch','trash','tune','zone','anger','award','bid','bitter','boot','bug',
    'camp','candy','carpet','cat','champion','channel','clock','comfort','cow','crack','engineer',
    'entrance','fault','grass','guy','hell','highlight','incident','island','joke','jury','leg','lip',
    'mate','motor','nerve','passage','pen','pride','priest','prize','promise','resident','resort','ring',
    'roof','rope','sail','scheme','script','sock','station','toe','tower','truck','witness','a','you',
    'it','can','will','if','one','many','most','other','use','make','good','look','help','go','great',
    'being','few','might','still','public','read','keep','start','give','human','local','general','she',
    'specific','long','play','feel','high','tonight','put','common','set','change','simple','past','big',
    'possible','particular','today','major','personal','current','national','cut','natural','physical',
    'show','try','check','second','call','move','pay','let','increase','single','individual','turn',
    'ask','buy','guard','hold','main','offer','potential','professional','international','travel','cook',
    'alternative','following','special','working','whole','dance','excuse','cold','commercial','low',
    'purchase','deal','primary','worth','fall','necessary','positive','produce','search','present',
    'spend','talk','creative','tell','cost','drive','green','support','glad','remove','return','run',
    'complex','due','effective','middle','regular','reserve','independent','leave','original','reach',
    'rest','serve','watch','beautiful','charge','active','break','negative','safe','stay','visit',
    'visual','affect','cover','report','rise','walk','white','beyond','junior','pick','unique',
    'anything','classic','final','lift','mix','private','stop','teach','western','concern','familiar',
    'fly','official','broad','comfortable','gain','maybe','rich','save','stand','young','fail','heavy',
    'hello','lead','listen','valuable','worry','handle','leading','meet','release','sell','finish',
    'normal','press','ride','secret','spread','spring','tough','wait','brown','deep','display','flow',
    'hit','objective','shoot','touch','cancel','chemical','cry','dump','extreme','push','conflict','eat',
    'fill','formal','jump','kick','opposite','pass','pitch','remote','total','treat','vast','abuse',
    'beat','burn','deposit','print','raise','sleep','somewhere','advance','anywhere','consist','dark',
    'double','draw','equal','fix','hire','internal','join','kill','sensitive','tap','win','attack',
    'claim','constant','drag','drink','guess','minor','pull','raw','soft','solid','wear','weird',
    'wonder','annual','count','dead','doubt','feed','forever','impress','nobody','repeat','round','sing',
    'slide','strip','whereas','wish','combine','command','dig','divide','equivalent','hang','hunt',
    'initial','march','mention','smell','spiritual','survey','tie','adult','brief','crazy','escape',
    'gather','hate','prior','repair','rough','sad','scratch','sick','strike','employ','external','hurt',
    'illegal','laugh','lay','mobile','nasty','ordinary','respond','royal','senior','split','strain',
    'struggle','swim','train','upper','wash','yellow','convert','crash','dependent','fold','funny',
    'grab','hide','miss','permit','quote','recover','resolve','roll','sink','slip','spare','suspect',
    'sweet','swing','twist','upstairs','usual','abroad','brave','calm','concentrate','estimate','grand',
    'male','mine','prompt','quiet','refuse','regret','reveal','rush','shake','shift','shine','steal',
    'suck','surround','anybody','bear','brilliant','dare','dear','delay','drunk','female','hurry',
    'inevitable','invite','kiss','neat','pop','punch','quit','reply','representative','resist','rip',
    'rub','silly','smile','spell','stretch','stupid','tear','temporary','tomorrow','wake','wrap',
    'yesterday']

def get_random_name(with_ext=True):
    return "{}_{}_{}{}".format(
        random.choice(adjectives),
        random.choice(nouns),
        random.randint(0, 50000),
        with_ext and '.txt' or '')

def get_random_file(max_filesize):
    file_start = random.randint(0, (max_filesize - 1025))
    file_size = random.randint(0, (max_filesize - file_start))
    file_name = get_random_name()
    return "{}:{}:{}".format(file_start, file_size, file_name)

def get_stream(name, max_filesize, data_loc, args):
    files = []
    for _ in range(random.randint(args.min_files, args.max_files)):
        files.append(get_random_file(max_filesize))
    stream = "{} {} {}".format(name, data_loc, ' '.join(files))
    return stream

def create_substreams(depth, base_stream_name, max_filesize, data_loc, args, current_size=0):
    current_stream = get_stream(base_stream_name, max_filesize, data_loc, args)
    current_size += len(current_stream)
    streams = [current_stream]

    if current_size >= max_manifest_size:
        logger.debug("Maximum manifest size reached -- finishing early at {}".format(base_stream_name))
    elif depth == 0:
        logger.debug("Finished stream {}".format(base_stream_name))
    else:
        for _ in range(random.randint(args.min_subdirs, args.max_subdirs)):
            stream_name = base_stream_name+'/'+get_random_name(False)
            substreams = create_substreams(depth-1, stream_name, max_filesize,
                data_loc, args, current_size)
            current_size += sum([len(x) for x in substreams])
            if current_size >= max_manifest_size:
                break
            streams.extend(substreams)
    return streams

def parse_arguments(arguments):
    args = arg_parser.parse_args(arguments)
    if args.debug:
        logger.setLevel(logging.DEBUG)
    if args.max_files < args.min_files:
        arg_parser.error("--min-files={} should be less or equal than max-files={}".format(args.min_files, args.max_files))
    if args.min_depth < 0:
        arg_parser.error("--min-depth should be at least 0")
    if args.max_depth < 0 or args.max_depth < args.min_depth:
        arg_parser.error("--max-depth should be at >= 0 and >= min-depth={}".format(args.min_depth))
    if args.max_subdirs < args.min_subdirs:
        arg_parser.error("--min-subdirs={} should be less or equal than max-subdirs={}".format(args.min_subdirs, args.max_subdirs))
    return args

def main(arguments=None):
    args = parse_arguments(arguments)
    logger.info("Creating test collection with (min={}, max={}) files per directory and a tree depth of (min={}, max={}) and (min={}, max={}) subdirs in each depth level...".format(args.min_files, args.max_files, args.min_depth, args.max_depth, args.min_subdirs, args.max_subdirs))
    api = arvados.api('v1', timeout=5*60)
    max_filesize = 1024*1024
    data_block = ''.join([random.choice(string.printable) for i in range(max_filesize)])
    data_loc = arvados.KeepClient(api).put(data_block)
    streams = create_substreams(random.randint(args.min_depth, args.max_depth),
        '.', max_filesize, data_loc, args)
    manifest = ''
    for s in streams:
        if len(manifest)+len(s) > max_manifest_size:
            logger.info("Skipping stream {} to avoid making a manifest bigger than 128MiB".format(s.split(' ')[0]))
            break
        manifest += s + '\n'
    try:
        coll_name = get_random_name(False)
        coll = api.collections().create(
            body={"collection": {
                "name": coll_name,
                "manifest_text": manifest
            },
        }).execute()
    except:
        logger.info("ERROR creating collection with name '{}' and manifest:\n'{}...'\nSize: {}".format(coll_name, manifest[0:1024], len(manifest)))
        raise
    logger.info("Created collection {} - manifest size: {}".format(coll["uuid"], len(manifest)))
    return 0

if __name__ == "__main__":
    sys.exit(main())