; Сапелин А.С., А-12-23, 18, v1
$INCLUDE (vars.inc)
DSEG AT 30h
CSEG AT 0h
        sjmp START
org 03h
        nop
        reti
org 0bh
        nop
        reti
org 13h
        nop
        reti
org 1bh
        nop
        reti
org 23h
        nop
        reti
org 2bh
START:
        mov SP, #07h            ; начальный стек по шаблону ТЗ
        mov IE, #0              ; самостоятельная программа не обслуживает IRQ
        setb PIN_CSEN           ; доступ к ОЗУ разрешается дешифратором
        mov HEAD_L, #LOW BUF_START ; следующая запись начнётся с начала кольца
        mov HEAD_H, #HIGH BUF_START
        mov TAIL_L, #LOW BUF_START ; чтение и запись пока имеют общий адрес
        mov TAIL_H, #HIGH BUF_START
        setb F_EMPTY            ; в памяти ещё нет действительных отсчётов
        clr F_OVF               ; исходное состояние не является переполнением
        call BufRead ; %proc%
        jmp $ ; %stop%

; Назначение: BufRead извлекает самый старый отсчёт из кольцевого буфера.
; Вход: HEAD, TAIL и флаги из vars.inc; данные во внешнем ОЗУ.
; Выход: A — прочитанный байт, при пустом буфере A = 0 без изменения указателей и флагов.
; Выход: после чтения TAIL передвинут, F_OVF = 0, F_EMPTY отражает опустошение.
; Портит: A, DPTR, флаги PSW; B сохраняется.
BufRead:
        jnb F_EMPTY, ReadData    ; пустой буфер нельзя читать и передвигать
        clr A
        ret
ReadData:
        push B                  ; временное хранение отсчёта не меняет B вызывающей программы
        mov DPL, TAIL_L
        mov DPH, TAIL_H          ; адрес самого старого отсчёта
        movx A, @DPTR
        mov B, A                ; сохранить результат на время проверки границ
        inc DPTR                ; предварительно перейти к следующему отсчёту
        mov A, DPL
        cjne A, #LOW BUF_END, ReadStore
        mov A, DPH
        cjne A, #HIGH BUF_END, ReadStore
        mov DPTR, #BUF_START     ; конец кольца переходит в его начало
ReadStore:
        mov TAIL_L, DPL
        mov TAIL_H, DPH          ; зафиксировать новый адрес чтения
        clr F_OVF               ; прочитанный отсчёт освободил место
        clr F_EMPTY
        mov A, DPL
        cjne A, HEAD_L, ReadDone ; разные указатели означают оставшиеся данные
        mov A, DPH
        cjne A, HEAD_H, ReadDone
        setb F_EMPTY            ; хвост догнал голову после последнего отсчёта
ReadDone:
        mov A, B                ; вернуть именно отсчёт, а не байт адреса
        pop B
        ret
END
