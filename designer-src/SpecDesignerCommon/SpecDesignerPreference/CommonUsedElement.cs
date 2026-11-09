using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.Xml.Linq;

namespace SpecDesignerPreference
{
	// Token: 0x02000087 RID: 135
	public class CommonUsedElement
	{
		// Token: 0x1700019D RID: 413
		// (get) Token: 0x06000598 RID: 1432 RVA: 0x00019D3B File Offset: 0x00017F3B
		public ReadOnlyObservableCollection<CommonUsedShortcut> EnableList
		{
			get
			{
				return this._enableBindingList;
			}
		}

		// Token: 0x1700019E RID: 414
		// (get) Token: 0x06000599 RID: 1433 RVA: 0x00019D43 File Offset: 0x00017F43
		public ReadOnlyObservableCollection<CommonUsedShortcut> DisableList
		{
			get
			{
				return this._disableBindingList;
			}
		}

		// Token: 0x0600059A RID: 1434 RVA: 0x00019D4B File Offset: 0x00017F4B
		public CommonUsedElement()
		{
			this._enableList = new ObservableCollection<CommonUsedShortcut>();
			this._enableBindingList = new ReadOnlyObservableCollection<CommonUsedShortcut>(this._enableList);
			this._disableList = new ObservableCollection<CommonUsedShortcut>();
			this._disableBindingList = new ReadOnlyObservableCollection<CommonUsedShortcut>(this._disableList);
		}

		// Token: 0x0600059B RID: 1435 RVA: 0x00019D8C File Offset: 0x00017F8C
		public static CommonUsedElement Parse(XElement source)
		{
			CommonUsedElement commonUsedElement = new CommonUsedElement();
			if (source.Element("CommonUsed") != null)
			{
				XElement xelement = source.Element("CommonUsed");
				foreach (XElement xelement2 in xelement.Elements("Item"))
				{
					CommonUsedShortcut commonUsedShortcut = CommonUsedShortcut.Parse(xelement2);
					commonUsedElement.Add(commonUsedShortcut);
				}
			}
			return commonUsedElement;
		}

		// Token: 0x0600059C RID: 1436 RVA: 0x00019E1C File Offset: 0x0001801C
		public XElement ToXML()
		{
			XElement xelement = new XElement("CommonUsed");
			foreach (CommonUsedShortcut commonUsedShortcut in this._enableList)
			{
				xelement.Add(commonUsedShortcut.ToXML());
			}
			foreach (CommonUsedShortcut commonUsedShortcut2 in this._disableList)
			{
				xelement.Add(commonUsedShortcut2.ToXML());
			}
			return xelement;
		}

		// Token: 0x0600059D RID: 1437 RVA: 0x00019EC8 File Offset: 0x000180C8
		public void Add(CommonUsedShortcut element)
		{
			if (element == null)
			{
				return;
			}
			if (element.Enabled == "Y")
			{
				this._enableList.Add(element);
				return;
			}
			this._disableList.Add(element);
		}

		// Token: 0x0600059E RID: 1438 RVA: 0x00019EF9 File Offset: 0x000180F9
		public void Move(CommonUsedShortcut element, int oldIndex, int newIndex)
		{
			if (element == null)
			{
				return;
			}
			if (element.Enabled == "Y")
			{
				return;
			}
			this._disableList.Insert(newIndex, element);
			this._disableList.RemoveAt(oldIndex);
		}

		// Token: 0x0600059F RID: 1439 RVA: 0x00019F2B File Offset: 0x0001812B
		public void Remove(CommonUsedShortcut element)
		{
			if (element == null)
			{
				return;
			}
			if (element.Enabled == "Y")
			{
				this._enableList.Remove(element);
				return;
			}
			this._disableList.Remove(element);
		}

		// Token: 0x060005A0 RID: 1440 RVA: 0x00019F5E File Offset: 0x0001815E
		internal void Clear()
		{
			this._enableList.Clear();
			this._disableList.Clear();
		}

		// Token: 0x1700019F RID: 415
		// (get) Token: 0x060005A1 RID: 1441 RVA: 0x0001A1C0 File Offset: 0x000183C0
		public IEnumerable<CommonUsedShortcut> Items
		{
			get
			{
				foreach (CommonUsedShortcut s in this._enableList)
				{
					yield return s;
				}
				foreach (CommonUsedShortcut s2 in this._disableList)
				{
					yield return s2;
				}
				yield break;
			}
		}

		// Token: 0x04000224 RID: 548
		private ObservableCollection<CommonUsedShortcut> _enableList;

		// Token: 0x04000225 RID: 549
		private ReadOnlyObservableCollection<CommonUsedShortcut> _enableBindingList;

		// Token: 0x04000226 RID: 550
		private ObservableCollection<CommonUsedShortcut> _disableList;

		// Token: 0x04000227 RID: 551
		private ReadOnlyObservableCollection<CommonUsedShortcut> _disableBindingList;
	}
}
