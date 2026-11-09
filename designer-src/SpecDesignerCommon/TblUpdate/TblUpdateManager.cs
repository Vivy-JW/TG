using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Net;

namespace SpecDesignerCommon.TblUpdate
{
	// Token: 0x0200001B RID: 27
	public class TblUpdateManager : IDisposable
	{
		// Token: 0x060000C5 RID: 197 RVA: 0x0000553C File Offset: 0x0000373C
		public TblUpdateManager(string localConfig, string castUrl)
		{
			this._castUrl = castUrl;
			this._localConfigPath = localConfig;
			this._configName = Path.GetFileName(this._castUrl);
			if (string.IsNullOrEmpty(this._castUrl) || string.IsNullOrEmpty(this._localConfigPath) || string.IsNullOrEmpty(this._configName))
			{
				return;
			}
			this._tempConfigName = string.Format("{0}{1}", Path.GetFileNameWithoutExtension(this._configName), ".tmp");
			this.StartUpdate();
		}

		// Token: 0x060000C6 RID: 198 RVA: 0x00005600 File Offset: 0x00003800
		public bool StartUpdate()
		{
			bool flag = true;
			try
			{
				this._content = this.GetUpdateContent();
				string lastUpdateContent = this.GetLastUpdateContent();
				this._castItem = TblUpdateCastItem.Parse(this._content);
				try
				{
					this._localItem = TblUpdateCastItem.Parse(lastUpdateContent);
				}
				catch
				{
					this._localItem = null;
				}
				this.GetUpdateItems(this._castItem, this._localItem);
				this.UpdateTbl();
				this.UpdateConfigFile();
			}
			catch
			{
				flag = false;
			}
			return flag;
		}

		// Token: 0x060000C7 RID: 199 RVA: 0x00005690 File Offset: 0x00003890
		private void UpdateConfigFile()
		{
			if (string.IsNullOrEmpty(this._content))
			{
				return;
			}
			string text = Path.Combine(this._localConfigPath, this._configName);
			File.WriteAllText(text, this._content);
			File.Delete(Path.Combine(this._localConfigPath, this._tempConfigName));
		}

		// Token: 0x060000C8 RID: 200 RVA: 0x00005700 File Offset: 0x00003900
		private void GetUpdateItems(TblUpdateCastItem compare, TblUpdateCastItem target)
		{
			IOrderedEnumerable<TblUpdateGroup> orderedEnumerable = compare.Items.OrderByDescending<TblUpdateGroup, DateTime>((TblUpdateGroup i) => i.Date);
			TblUpdateGroup g;
			foreach (TblUpdateGroup tblUpdateGroup in orderedEnumerable)
			{
				g = tblUpdateGroup;
				if (target != null)
				{
					int num = target.Items.Where<TblUpdateGroup>((TblUpdateGroup i) => i.CompareTo(g) == 0).Count<TblUpdateGroup>();
					if (num > 0)
					{
						break;
					}
				}
				foreach (TblUpdateItem tblUpdateItem in g.Items)
				{
					if (tblUpdateItem.Lang == Lang.zh_TW)
					{
						this._needUpdateItem.Add(tblUpdateItem);
					}
				}
			}
		}

		// Token: 0x060000C9 RID: 201 RVA: 0x00005810 File Offset: 0x00003A10
		private void UpdateTbl()
		{
			if (this._castItem == null)
			{
				return;
			}
			string text = this._castUrl.Remove(this._castUrl.LastIndexOf(this._configName));
			foreach (TblUpdateItem tblUpdateItem in this._needUpdateItem.Distinct<TblUpdateItem>())
			{
				string text2 = Path.Combine(text, tblUpdateItem.RemotePath);
				string text3 = Path.Combine(this._localConfigPath, tblUpdateItem.LocalPath);
				if (!TblUpdateManager.DownloadFile(text2, text3))
				{
					this._content = string.Empty;
					break;
				}
			}
		}

		// Token: 0x060000CA RID: 202 RVA: 0x000058BC File Offset: 0x00003ABC
		private string GetUpdateContent()
		{
			string text = Path.Combine(this._localConfigPath, this._tempConfigName);
			TblUpdateManager.DownloadFile(this._castUrl, text);
			string text2 = string.Empty;
			if (File.Exists(text))
			{
				text2 = File.ReadAllText(text);
			}
			return text2;
		}

		// Token: 0x060000CB RID: 203 RVA: 0x00005900 File Offset: 0x00003B00
		private string GetLastUpdateContent()
		{
			string text = string.Empty;
			if (File.Exists(this._localConfigPath))
			{
				text = File.ReadAllText(this._localConfigPath);
			}
			return text;
		}

		// Token: 0x060000CC RID: 204 RVA: 0x00005930 File Offset: 0x00003B30
		internal static bool DownloadFile(string url, string localPath)
		{
			byte[] array = null;
			byte[] array2 = new byte[4097];
			Stream stream = null;
			MemoryStream memoryStream = null;
			try
			{
				WebRequest webRequest = WebRequest.Create(url);
				webRequest.Timeout = TblUpdateManager.REQUEST_TIMEOUT;
				WebResponse response = webRequest.GetResponse();
				stream = response.GetResponseStream();
				memoryStream = new MemoryStream();
				int num;
				do
				{
					num = stream.Read(array2, 0, array2.Length);
					memoryStream.Write(array2, 0, num);
				}
				while (num != 0);
				array = memoryStream.ToArray();
			}
			catch
			{
				return false;
			}
			FileStream fileStream = new FileStream(localPath, FileMode.OpenOrCreate, FileAccess.ReadWrite);
			fileStream.SetLength(0L);
			fileStream.Write(array, 0, array.Length);
			fileStream.Close();
			memoryStream.Close();
			stream.Close();
			return true;
		}

		// Token: 0x060000CD RID: 205 RVA: 0x000059EC File Offset: 0x00003BEC
		public void Dispose()
		{
			if (this._needUpdateItem != null)
			{
				this._needUpdateItem.Clear();
			}
			if (this._castItem != null)
			{
				this._castItem.Items.Clear();
			}
			if (this._localItem != null)
			{
				this._localItem.Items.Clear();
			}
		}

		// Token: 0x060000CE RID: 206 RVA: 0x00005C00 File Offset: 0x00003E00
		public IEnumerable<string> GetNeddUpdateItems()
		{
			if (this._needUpdateItem == null)
			{
				yield return null;
			}
			else
			{
				foreach (TblUpdateItem tui in this._needUpdateItem)
				{
					yield return tui.Name;
				}
			}
			yield break;
		}

		// Token: 0x04000049 RID: 73
		private string _localConfigPath = "C:\\TT\\";

		// Token: 0x0400004A RID: 74
		private string _configName = "AlterTableList.xml";

		// Token: 0x0400004B RID: 75
		private string _tempConfigName = "AlterTableList.tmp";

		// Token: 0x0400004C RID: 76
		private List<TblUpdateItem> _needUpdateItem = new List<TblUpdateItem>();

		// Token: 0x0400004D RID: 77
		private string _castUrl = string.Empty;

		// Token: 0x0400004E RID: 78
		private string _content = string.Empty;

		// Token: 0x0400004F RID: 79
		private TblUpdateCastItem _castItem;

		// Token: 0x04000050 RID: 80
		private TblUpdateCastItem _localItem;

		// Token: 0x04000051 RID: 81
		private static readonly int REQUEST_TIMEOUT = 5000;
	}
}
