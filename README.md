# Diplomski rad: *Protočna obrada podataka u programskom jeziku Go*

## Motivacija

Go je jezik koji podržava CSP (Communicating Sequential Processes) formalizam za opis interakcije konkurentnih sistema. Na osnovu ovog formalizma, u jezik su uvedeni koncepti kanala i gorutina. Ovi koncepti omogućuju pisanje konkurentnog koda koji apstrahuje korišćenje tradicionalnih primitiva za sinhronizaciju pristupa deljenoj memoriji (muteksi, semafori).

Šablon protočne obrade ili "Pipe and Filter" je šablon generalno prisutan u računarskoj nauci kojeg karakteriše efikasna kaskadna obrada podataka u više etapa.

Koncepti konkurentnog programiranja dostupni u jeziku Go pogodni su za implementaciju ovakvog šablona. 

Primeri šablona protočne obrade predstavljeni na zvaničnim izvorima o Go jeziku su simbolični. Stiče se utisak da su predstavljeni kao "proof-of-concept" primene Go-ovog konkurentnog modela.
Štaviše, nigde nisu predstavljeni podaci o poboljšanju performansi pri primeni ovog šablona, niti je analizirana algoritamska složenost kada se primenjuju i podšabloni fan-out, fan-in, rate-limiting.

## Cilj rada
Prikazati mogućnosti protočne obrade podataka u jeziku Go. Primeniti serijsku i protočnu obradu podataka na primeru iz prakse. Analizirati performanse serijske i protočne obrade podataka.

## Sadržaj rada

Rad bi imao tri velika dela: (Izlistani su delovi i pojašnjen je njihov sadržaj)

1. Teorijska osnova (recimo 10% sadrzaja)
        
    - CSP i Go-ova podrška za CSP (objašnjenje koncepata poput kanala (chan <T>), read kanala (<-chan <T>) i write kanala (chan<- <T>))
    - Šablon protočne obrade
2. Primena (65%)
    - Definicija primera iz prakse
    
      Digitalna galerija - jednostavna i performantna platforma na kojoj korisnik može da uploaduje veliki broj slika i potom da ih pregleda. Kako bi podržala veliki broj slika, pri implementaciji platforme su korišćeni koncepti programskog jezika Go za pisanje konkurentnog koda i protočna obrada podataka.
        - Lista zahteva digitalne galerije, REST API definicija i izgled frontend klijenta
        - Model faza protočne obrade (objašnjene su faze obrade slika pri uploadu slika - [**dokument1**](serijska_obrada_slika.pdf))
    - Primena šablona protočne obrade
        - Serijska obrada (osnovni primer, [**dokument1**](serijska_obrada_slika.pdf))
        - Osnovna protočna obrada (faze su ekvivalentne serijskim filterima u Intel TBB - [**dokument2**](osnovna_protocna_obrada.pdf))
        - Paralelna protočna obrada 
          
          Primena Fan-out, Fan-in podšablona, faze su sad ekvivalentne paralelnim filterima, slike su obrađene u slučajnom rasporedu)
        - Ograničena paralelna protočna obrada (Primena Rate-limiting šablona - Ograničavamo paralelnu obradu na npr. 30 slika u fazi Resize kako bismo ograničili broj gorutina
3. Analiza performansi i vizualizacija (u jeziku Pharo, koristeći biblioteku Roassal) (25%)
    - Vizualizacija performansi obrade slika 
    
      Vizualizacija performansi zahteva ka endpointu za upload slika kod serijske obrade i osnovne, paralelne i ograničene protočne obrade slika (ili manuelno ili apache-benchmark). Neke od metrika koje će biti uključene: Brzina obrade zahteva, Ukupan broj gorutina, Ukupno memorijsko zauzeće i još neke koje bih kasnije odredio.
    - Analiza performansi

## Literatura

Planiram najviše da se oslonim na sledeće dve knjige: 
1. The Go Programming Language (poglavlje 8) 
2. Concurrency in Go (poglavlje 4 - Pipeline pattern).

## Generalan doprinos rada oblasti

Smatram da je najveći doprinos rada prikaz primera iz prakse nad kojim je primenjena protočna obrada. Ovaj primer će biti većeg "obima" od primera koji se mogu naći na zvaničnim izvorima o programskom jeziku Go i realan je. 

Takođe, nisam našao materijal u kom su analizirane performanse pri primeni ovog šablona ili u kom su poređene performanse serijske i različitih protočnih obrada podataka.

## Plan rada
Planiram da do odbrane projekta implementiram serijsku obradu, kao i osnovnu protočnu obradu, a ukoliko vreme dozvoli i ostale dve. Takođe, kreirao bih i Pharo servis koristeći Roassal biblioteku koja bi vizualizovala navedene metrike. Za kasniji rad bih ostavio neodrađene šablone protočne obrade, proširenu analizu performansi i složenosti i pisanje teorije.


# Pokretanje
1. Baza podataka

`docker run --name go-pipelines-postgres -e POSTGRES_PASSWORD=go-pipelines -e POSTGRES_USER=go-pipelines -e POSTGRES_DB=go-pipelines -d -p 5432:5432 postgres`
`docker exec -it go-pipelines-postgres bash`
`psql -U go-pipelines`

```sql
CREATE TABLE image (id serial PRIMARY KEY, name VARCHAR, fullpath VARCHAR, thumbnailpath VARCHAR);
```
```sql
CREATE TABLE "user" (id serial PRIMARY KEY);
```
```sql
CREATE TABLE user_images (user_id INT NOT NULL, image_id INT NOT NULL, PRIMARY KEY (user_id, image_id), FOREIGN KEY (user_id) REFERENCES "user"(id), FOREIGN KEY (image_id) REFERENCES image(id));
```
