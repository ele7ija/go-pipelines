# Diplomski rad: *Protočna obrada podataka u programskom jeziku Go*

## Motivacija

Go je jezik koji podržava CSP (Communicating Sequential Processes) formalizam za opis interakcije konkurentnih sistema. Na osnovu ovog formalizma, u jezik su uvedeni koncepti kanala i gorutina. Ovi koncepti omogućuju pisanje konkurentnog koda koji apstrahuje korišćenje tradicionalnih primitiva za sinhronizaciju pristupa deljenoj memoriji (muteksi, semafori).

Šablon protočne obrade ili "Pipe and Filter" je šablon generalno prisutan u računarskoj nauci kojeg karakteriše efikasna kaskadna obrada podataka u više etapa.

Koncepti konkurentnog programiranja dostupni u jeziku Go pogodni su za implementaciju ovakvog šablona.

Ne postoji implementacija protočne obrade u formi paketa koja je u širokoj upotrebi.
Primeri šablona protočne obrade predstavljeni na zvaničnim izvorima o Go jeziku su simbolični i mali po obimu.
Nigde nisu predstavljeni podrobni podaci o poboljšanju performansi pri primeni ovog šablona.

## Cilj rada
Implementirati protočnu obradu podataka u jeziku Go. 
Primeniti protočnu obradu podataka na primeru iz prakse. 
Analizirati performanse protočne obrade podataka.
**Prilagoditi i izdati u slobodnu upotrebu paket za protočnu obradu**.

## Sadržaj rada

Rad bi imao tri velika dela: (Izlistani su delovi i pojašnjen je njihov sadržaj)

1. Teorijska osnova (recimo 10% sadrzaja)

    1. CSP i Go-ova podrška za CSP (objašnjenje koncepata poput kanala (chan <T>), read kanala (<-chan <T>) i write kanala (chan<- <T>))
    2. Šablon protočne obrade (šta je protočna obrada, šta je filter)
2. Primena (50%)
   1. Implementacija protočne obrade u Go-u
   
      Objašnjeni delovi koda iz paketa za protočnu obradu. Verifikacija uz pomoć Go jediničnih testova.
   2. Definicija primera iz prakse

      Digitalna galerija - jednostavna i performantna platforma na kojoj korisnik može da uploaduje veliki broj slika
      i potom da ih pregleda. Kako bi podržala veliki broj slika, pri implementaciji platforme je korišćen paket za protočnu obradu podataka.
        - Lista zahteva digitalne galerije, REST API definicija i izgled frontend klijenta
        - Faze protočne obrade (objašnjene su faze obrade slika pri uploadu slika - [**dokument1**](assets/serijska_obrada_slika.pdf))
   3. Primenjene protočne obrade
      
      Koje obrade (kombinacije filtera tačnije) će biti uzete u obzir i zašto.
        - Serijska obrada (osnovni primer, [**dokument1**](assets/serijska_obrada_slika.pdf))
        - Osnovna protočna obrada (faze su ekvivalentne serijskim filterima u Intel TBB - [**dokument2**](assets/osnovna_protocna_obrada.pdf))
        - Paralelna protočna obrada
        - Ograničena paralelna protočna obrada (Primena Rate-limiting šablona - Ograničavamo paralelnu obradu na npr. 30 slika u fazi Resize kako bismo ograničili zauzeće resursa)
3. Eksperimenti, rezultati i komentari (40%) - poput [ovog dokumenta](https://github.com/ele7ija/go-pipelines/blob/main/Rezultati.md#rezultati-obrada)
   1. Opis eksperimenata
   2. Rezultati
      
      Vizualizacija rezultata. Poređenje, analiza, glavni zaključci. 
   3. Komentari
      
      Domensko znanje koje sam stekao prilikom primene protočne obrade i analize rezultata.

## Literatura

Planiram najviše da se oslonim na sledeće dve knjige:
1. The Go Programming Language (poglavlje 8)
2. Concurrency in Go (poglavlje 4 - Pipeline pattern).

## Generalan doprinos rada oblasti

Smatram da su najveći doprinosi rada:
 - Paket za protočnu obradu koji će biti publikovan
 - Prikaz primera iz prakse nad kojim je primenjena protočna obrada i rezultata. Ovaj primer će biti većeg "obima" od primera koji se mogu naći na zvaničnim izvorima o programskom jeziku Go i realan je.