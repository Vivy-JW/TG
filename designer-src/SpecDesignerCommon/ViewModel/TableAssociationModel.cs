using System;
using System.Collections.Generic;
using System.Linq;
using System.Xml.Linq;
using SpecDesignerCommon.Helpers;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x0200013D RID: 317
	public class TableAssociationModel : IDisposable
	{
		// Token: 0x170002E5 RID: 741
		// (get) Token: 0x06000B0B RID: 2827 RVA: 0x000357A9 File Offset: 0x000339A9
		// (set) Token: 0x06000B0C RID: 2828 RVA: 0x000357B1 File Offset: 0x000339B1
		public PackageKey Key { get; private set; }

		// Token: 0x170002E6 RID: 742
		// (get) Token: 0x06000B0D RID: 2829 RVA: 0x000357BA File Offset: 0x000339BA
		// (set) Token: 0x06000B0E RID: 2830 RVA: 0x000357C2 File Offset: 0x000339C2
		public XElement Source { get; private set; }

		// Token: 0x06000B0F RID: 2831 RVA: 0x000357CB File Offset: 0x000339CB
		public TableAssociationModel(PackageKey key, XElement xml)
		{
			this.Source = xml;
			this.Key = key;
			this.createTBLModel(xml);
		}

		// Token: 0x170002E7 RID: 743
		// (get) Token: 0x06000B10 RID: 2832 RVA: 0x000357F3 File Offset: 0x000339F3
		public List<TBLModel> AliveTBLs
		{
			get
			{
				return this.tbls.ToList<TBLModel>();
			}
		}

		// Token: 0x06000B11 RID: 2833 RVA: 0x00035800 File Offset: 0x00033A00
		public TBLModel CreateNewTBLModel(string tblName, string parent)
		{
			XElement xelement = XElement.Parse("<tbl name='' head='N' pk='' parent='' main='N' src='' status='c'/>", LoadOptions.None);
			xelement.SetAttributeValue("name", tblName);
			this.Source.Add(xelement);
			if (!string.IsNullOrEmpty(parent))
			{
				xelement.SetAttributeValue("parent", parent);
			}
			TBLModel tblmodel = new TBLModel(xelement, this);
			this.tbls.Add(tblmodel);
			return tblmodel;
		}

		// Token: 0x06000B12 RID: 2834 RVA: 0x00035864 File Offset: 0x00033A64
		public void Dispose()
		{
			foreach (TBLModel tblmodel in this.tbls)
			{
				tblmodel.Dispose();
			}
			this.tbls.Clear();
			this.Source = null;
		}

		// Token: 0x06000B13 RID: 2835 RVA: 0x000358C8 File Offset: 0x00033AC8
		private void createTBLModel(XElement xml)
		{
			foreach (XElement xelement in xml.Elements("tbl"))
			{
				if (xelement.Attribute("status") != null && xelement.Attribute("status").Value != ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE))
				{
					this.tbls.Add(new TBLModel(xelement, this));
				}
			}
		}

		// Token: 0x06000B14 RID: 2836 RVA: 0x00035964 File Offset: 0x00033B64
		internal void Remove(string componentName)
		{
			foreach (TBLModel tblmodel in this.tbls)
			{
				tblmodel.Remove(componentName);
			}
		}

		// Token: 0x06000B15 RID: 2837 RVA: 0x000359B8 File Offset: 0x00033BB8
		public void Remove(TBLModel delModel)
		{
			if (delModel == null)
			{
				return;
			}
			delModel.Status = ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE);
			delModel.XML.Remove();
			this.Source.AddFirst(delModel.XML);
		}

		// Token: 0x04000442 RID: 1090
		internal List<TBLModel> tbls = new List<TBLModel>();
	}
}
