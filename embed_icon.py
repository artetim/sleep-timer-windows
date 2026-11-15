"""Embed a standard multi-resolution ICO into a Windows PE32+ executable.
Pure Python stdlib. Usage: python embed_icon.py input.exe SleepTimer.ico output.exe
Designed for unsigned Go Windows builds without an existing .rsrc section.
"""
import struct
import sys
from pathlib import Path


def read16(buf, offset):return struct.unpack_from('<H', buf, offset)[0]
def read32(buf, offset):return struct.unpack_from('<I', buf, offset)[0]
def write16(buf, offset, value):struct.pack_into('<H', buf, offset, value)
def write32(buf, offset, value):struct.pack_into('<I', buf, offset, value)
def align(value, num):return (value+num-1)&~(num-1)


def read_ico(filename):
    data=Path(filename).read_bytes()
    reserved,kind,count=struct.unpack_from('<HHH',data,0)
    if reserved!=0 or kind!=1 or not 1<=count<=100:
        raise ValueError('Expected a standard .ico with 1 to 100 images')
    icons=[]
    for i in range(count):
        width,height,colors,reserved,planes,bits,size,offset=struct.unpack_from('<BBBBHHII',data,6+16*i)
        if offset+size>len(data) or size==0:
            raise ValueError('Invalid .ico image offset')
        icons.append((width,height,colors,reserved,planes,bits,data[offset:offset+size]))
    return icons


def resource_block(icons, base_rva):
    count=len(icons)
    b=bytearray()
    def reserve(length):
        offset=len(b);b.extend(b'\0'*length);return offset
    def directory(offset, entries):
        # All entries are by numeric ID, no name entries, no timestamps.
        struct.pack_into('<IIHHHH',b,offset,0,0,0,0,0,len(entries))
        for index,(rid,entry,subdir) in enumerate(entries):
            struct.pack_into('<II',b,offset+16+index*8,rid,entry | (0x80000000 if subdir else 0))
    root=reserve(16+2*8)
    icons_dir=reserve(16+count*8)
    groups_dir=reserve(16+8)
    ico_lang=[reserve(16+8) for _ in icons]
    grp_lang=reserve(16+8)
    ico_entries=[reserve(16) for _ in icons]
    grp_entry=reserve(16)
    b.extend(b'\0'*(align(len(b),4)-len(b)))
    image_offsets=[]
    for icon in icons:
        image_offsets.append(len(b))
        b.extend(icon[6]);b.extend(b'\0'*(align(len(b),4)-len(b)))
    group=bytearray(struct.pack('<HHH',0,1,count))
    for i,(w,h,colors,reserved,planes,bits,data) in enumerate(icons):
        group.extend(struct.pack('<BBBBHHIH',w,h,colors,reserved,planes,bits,len(data),i+1))
    group_offset=len(b)
    b.extend(group)
    b.extend(b'\0'*(align(len(b),4)-len(b)))
    directory(root,[(3,icons_dir,True),(14,groups_dir,True)])
    directory(icons_dir,[(i+1,ico_lang[i],True) for i in range(count)])
    directory(groups_dir,[(1,grp_lang,True)])
    for i in range(count):
        directory(ico_lang[i],[(0x409,ico_entries[i],False)])
        struct.pack_into('<IIII',b,ico_entries[i],base_rva+image_offsets[i],len(icons[i][6]),0,0)
    directory(grp_lang,[(0x409,grp_entry,False)])
    struct.pack_into('<IIII',b,grp_entry,base_rva+group_offset,len(group),0,0)
    return bytes(b)


def inject(exe_path,ico_path,dest_path):
    d=bytearray(Path(exe_path).read_bytes())
    pe=read32(d,0x3c)
    if d[pe:pe+4]!=b'PE\0\0' or read16(d,pe+4)!=0x8664:
        raise ValueError('Expected Windows x64 PE executable')
    n=read16(d,pe+6)
    oh=pe+24
    if read16(d,oh)!=0x20b:raise ValueError('Expected PE32+ format')
    sh=oh+read16(d,pe+20)
    file_align=read32(d,oh+36)
    section_align=read32(d,oh+32)
    if sh+40*(n+1)>read32(d,oh+60):raise ValueError('Insufficient section header space')
    if read32(d,oh+112+16)!=0:raise ValueError('Executable already has Windows resources')
    last_va=0
    for i in range(n):
        off=sh+40*i
        va=read32(d,off+12)
        virtual_size=read32(d,off+8)
        raw_size=read32(d,off+16)
        last_va=max(last_va,va+align(max(virtual_size,raw_size),section_align))
    new_va=align(last_va,section_align)
    block=resource_block(read_ico(ico_path),new_va)
    raw_size=align(len(block),file_align)
    raw_offset=align(len(d),file_align)
    d.extend(b'\0'*(raw_offset-len(d)))
    d.extend(block)
    d.extend(b'\0'*(raw_size-len(block)))
    off=sh+40*n
    d[off:off+40]=b'.rsrc\0\0\0'+struct.pack('<IIIIIIHHI',len(block),new_va,raw_size,raw_offset,0,0,0,0,0x40000040)
    write16(d,pe+6,n+1)
    write32(d,oh+56,align(new_va+len(block),section_align)) # SizeOfImage
    write32(d,oh+112+16,new_va)
    write32(d,oh+112+20,len(block))
    write32(d,oh+64,0) # Unsigned PE image. Windows checksum is not required.
    Path(dest_path).write_bytes(d)
    print(f'Embedded {len(read_ico(ico_path))} icon sizes into {dest_path}')


if __name__=='__main__':
    if len(sys.argv)!=4:raise SystemExit('Usage: python embed_icon.py input.exe icon.ico output.exe')
    inject(*sys.argv[1:])
