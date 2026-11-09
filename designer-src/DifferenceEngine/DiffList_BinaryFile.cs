using System;
using System.IO;

namespace DifferenceEngine
{
	// Token: 0x02000006 RID: 6
	public class DiffList_BinaryFile : IDiffList
	{
		// Token: 0x0600000D RID: 13 RVA: 0x00002210 File Offset: 0x00001210
		public DiffList_BinaryFile(string fileName)
		{
			FileStream fileStream = null;
			BinaryReader binaryReader = null;
			try
			{
				fileStream = new FileStream(fileName, FileMode.Open, FileAccess.Read);
				int num = (int)fileStream.Length;
				binaryReader = new BinaryReader(fileStream);
				this._byteList = binaryReader.ReadBytes(num);
			}
			catch (Exception ex)
			{
				throw ex;
			}
			finally
			{
				if (binaryReader != null)
				{
					binaryReader.Close();
				}
				if (fileStream != null)
				{
					fileStream.Close();
				}
			}
		}

		// Token: 0x0600000E RID: 14 RVA: 0x00002284 File Offset: 0x00001284
		public int Count()
		{
			return this._byteList.Length;
		}

		// Token: 0x0600000F RID: 15 RVA: 0x0000228E File Offset: 0x0000128E
		public IComparable GetByIndex(int index)
		{
			return this._byteList[index];
		}

		// Token: 0x04000008 RID: 8
		private byte[] _byteList;
	}
}
